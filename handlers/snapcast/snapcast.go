package snapcast

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
	"golang.org/x/net/context"
)

var logger = log.With().Str("handler", "snapcast").Logger()

type SnapcastClientCommand struct {
	ClientID string       `json:"client_id"`       // client id
	Action   types.Action `json:"action"`          // SET sets a value or unmutes it, RESET mutes the volume
	Value    int          `json:"value,omitempty"` // optional value
}

type SnapcastGroupCommand struct {
	GroupID string       `json:"group_id"`
	Action  types.Action `json:"action"`            // SET sets given options GET gets the group description
	Stream  string       `json:"stream,omitempty"`  // stream to set (valid only for SET)
	Clients []string     `json:"clients,omitempty"` // clients to group into this one (valid only for SET)
}

type snapcast struct {
	chik.BaseHandler
	client *jrpc2.Client
	status *chik.StatusHolder
	events chan interface{}
}

func New() chik.Handler {
	return &snapcast{
		status: chik.NewStatusHolder("snapcast"),
		events: make(chan interface{}),
	}
}

func (h *snapcast) Setup(controller *chik.Controller) (i chik.Interrupts, err error) {
	i = chik.Interrupts{
		Timer: chik.NewTimer(3*time.Second, true),
		Event: h.events,
	}
	return
}

func (h *snapcast) Teardown() {
	if h.client != nil {
		h.client.Close()
		h.client = nil
	}
}

func (h *snapcast) Topics() []types.CommandType {
	return []types.CommandType{
		types.SnapcastManagerCommandType,
		types.SnapcastClientCommandType,
		types.SnapcastGroupCommandType,
	}
}

func (h *snapcast) notify(req *jrpc2.Request) {
	h.events <- req
}

func (h *snapcast) getServerStatus(controller *chik.Controller) (err error) {
	resp, err := h.snapcastRequest(controller, "Server.GetStatus", nil)
	if err != nil {
		logger.Err(err).Msg("status request error")
		return
	}

	var serverStatus SimplifiedStatus
	err = resp.UnmarshalResult(&serverStatus)
	if err != nil {
		return err
	}

	rawStatus := h.status.Get()
	if rawStatus == nil {
		h.status.Set(serverStatus.Server, controller)
		return
	}
	currentStatus := rawStatus.(SimplifiedServerStatus)
	// if currentStatus has group stream changes
	// set the status of snapcast to our current status
	for _, group := range currentStatus.Groups {
		idx := slices.IndexFunc(currentStatus.Groups, func(g SimplifiedGroup) bool {
			return g.ID == group.ID
		})
		if idx == -1 {
			continue
		}
		if group.StreamID != currentStatus.Groups[idx].StreamID {
			h.snapcastRequest(controller, "Group.SetStream", map[string]interface{}{
				"id":        group.ID,
				"stream_id": currentStatus.Groups[idx].StreamID,
			})
		}
	}
	return
}

func (h *snapcast) isServerAlive(controller *chik.Controller) bool {
	_, err := h.snapcastRequest(controller, "Server.GetRPCVersion", nil)
	return err == nil
}

func (h *snapcast) connect(controller *chik.Controller) error {
	if h.client != nil {
		h.client.Close()
	}

	conn, err := net.Dial("tcp", "127.0.0.1:1705")
	if err != nil {
		return err
	}

	h.client = jrpc2.NewClient(channel.Line(conn, conn), &jrpc2.ClientOptions{
		OnNotify: h.notify,
	})

	return h.getServerStatus(controller)
}

func (h *snapcast) snapcastRequest(controller *chik.Controller, method string, params interface{}) (resp *jrpc2.Response, err error) {
	logger.Debug().Str("method", method).Msg("sending a snapcast request")
	if h.client == nil {
		logger.Info().Msg("connecting to server")
		if err = h.connect(controller); err != nil {
			logger.Err(err).Msg("connection failed")
			return
		}
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancel()

	resp, err = h.client.Call(ctx, method, params)
	return
}

func (h *snapcast) HandleTimerEvent(tick time.Time, controller *chik.Controller) (err error) {
	if h.client == nil || !h.isServerAlive(controller) {
		h.connect(controller)
	}

	return nil
}

func (h *snapcast) handleServerStatusUpdate(status *Status, controller *chik.Controller) {
	h.status.Set(status.Server, controller)
	return
}

func (h *snapcast) handleGroupStreamChanged(groupStream *GroupStreamChanged, controller *chik.Controller) {
	h.status.Edit(controller, func(current interface{}) (interface{}, bool) {
		status := current.(SimplifiedServerStatus)
		forceChange := false
		for gidx, group := range status.Groups {
			if group.ID == groupStream.ID {
				status.Groups[gidx].StreamID = groupStream.StreamID
				forceChange = true
			}
		}
		return status, forceChange
	})
}

func (h *snapcast) HandleChannelEvent(event interface{}, controller *chik.Controller) (err error) {
	req, ok := event.(*jrpc2.Request)
	if !ok {
		logger.Error().Msgf("unexpected channel event %v", req)
		return errors.New("Unexpected channel event")
	}
	logger.Info().Str("method", req.Method()).Msg("Notification received")

	switch req.Method() {
	case "Server.OnUpdate":
		var status Status
		err = req.UnmarshalParams(&status)
		if err != nil {
			logger.Err(err).Msg("notify decode failed")
			return
		}
		h.handleServerStatusUpdate(&status, controller)

	case "Group.OnStreamChanged":
		var groupStream GroupStreamChanged
		err = req.UnmarshalParams(&groupStream)
		if err != nil {
			logger.Err(err).Msg("Failed to decode group stream changed notification")
			return
		}
		h.handleGroupStreamChanged(&groupStream, controller)
	}
	return
}

func (h *snapcast) handleClientCommand(message *chik.Message, controller *chik.Controller) (err error) {
	var command SnapcastClientCommand
	err = json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Err(err).Msg("failed to decode client command")
		return
	}

	var volume Volume

	switch command.Action {
	case types.RESET:
		volume = Volume{Muted: true}
	case types.SET:
		volume = Volume{Muted: false, Percent: command.Value}
	default:
		logger.Error().Msgf("Unknown action %v", command.Action)
		return
	}
	h.snapcastRequest(controller, "Client.SetVolume", map[string]interface{}{"id": command.ClientID, "volume": volume})
	return
}

func (h *snapcast) handleGroupCommand(message *chik.Message, controller *chik.Controller) (err error) {
	var command SnapcastGroupCommand
	err = json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Err(err).Msg("failed to decode client command")
		return
	}
	logger.Debug().Msgf("Group command: %v", command)
	switch command.Action {
	case types.SET:
		if command.Stream != "" {
			var reply *jrpc2.Response
			reply, err = h.snapcastRequest(controller, "Group.SetStream", map[string]interface{}{"id": command.GroupID, "stream_id": command.Stream})
			if err != nil {
				return
			}
			var groupStream GroupStreamChanged
			err = reply.UnmarshalResult(&groupStream)
			if err != nil {
				return
			}
			groupStream.ID = command.GroupID
			h.handleGroupStreamChanged(&groupStream, controller)
		}

		if len(command.Clients) > 0 {
			h.snapcastRequest(controller, "Group.SetClients", map[string]interface{}{"id": command.GroupID, "clients": command.Clients})
		}
	case types.GET:
		var reply *jrpc2.Response
		reply, err = h.snapcastRequest(controller, "Server.GetStatus", nil)
		if err != nil {
			logger.Err(err).Msg("Failed to get server status")
			return
		}
		var status Status
		err = reply.UnmarshalResult(&status)
		if err != nil {
			return
		}
		h.handleServerStatusUpdate(&status, controller)
	}

	return
}

func (h *snapcast) HandleMessage(message *chik.Message, controller *chik.Controller) (err error) {
	logger.Debug().Msg("message received")
	switch message.Command().Type {
	case types.SnapcastClientCommandType:
		err = h.handleClientCommand(message, controller)
	case types.SnapcastGroupCommandType:
		err = h.handleGroupCommand(message, controller)
	}
	return
}

func (h *snapcast) String() string {
	return "snapcast"
}
