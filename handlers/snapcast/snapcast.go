package snapcast

import (
	"encoding/json"
	"net"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
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
		Timer: chik.NewTimer(60*time.Second, true),
		Event: h.events,
	}
	return
}

func (h *snapcast) Teardown() {
	if h.client != nil {
		h.client.Close()
	}
}

func (h *snapcast) Topics() []types.CommandType {
	return []types.CommandType{
		types.SnapcastManagerCommandType,
		types.SnapcastClientCommandType,
		types.SnapcastGroupCommandType,
	}
}

func (h *snapcast) setStatus(resp *jrpc2.Response, controller *chik.Controller) error {
	logger.Debug().Interface("raw_reply", resp)
	var serverStatus Status
	err := resp.UnmarshalResult(&serverStatus)
	if err != nil {
		return err
	}

	h.status.Set(serverStatus.Server, controller)
	return nil
}

func (h *snapcast) notify(req *jrpc2.Request) {
	h.events <- req
}

func (h *snapcast) connect() error {
	if h.client != nil {
		h.client.Close()
	}

	conn, err := net.Dial("tcp", "127.0.0.1:1705")
	if err != nil {
		return err
	}

	h.client = jrpc2.NewClient(LineJSON(conn, conn), &jrpc2.ClientOptions{
		OnNotify: h.notify,
	})
	return nil
}

func (h *snapcast) snapcastRequest(method string, params interface{}) (resp *jrpc2.Response, err error) {
	logger.Debug().Str("method", method).Msg("sending a snapcast request")
	if h.client == nil {
		logger.Info().Msg("connecting to server")
		if h.client != nil {
			h.client.Close()
		}
		if err = h.connect(); err != nil {
			logger.Err(err).Msg("connection failed")
			return
		}
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))
	defer cancel()

	resp, err = h.client.Call(ctx, method, params)
	if err != nil {
		logger.Err(err).Msg("call failed")
		logger.Info().Msg("attempting another iteration of call")
		h.client.Close()
		h.client = nil
		h.snapcastRequest(method, params)
	}

	return
}

func (h *snapcast) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	if h.client != nil {
		return
	}

	if err := h.connect(); err != nil {
		logger.Err(err).Msg("Server connection failed")
		return
	}

	logger.Debug().Msg("Timer event received")
	resp, err := h.snapcastRequest("Server.GetStatus", nil)
	if err != nil {
		logger.Err(err).Msg("status request error")
		return
	}

	if err := h.setStatus(resp, controller); err != nil {
		logger.Err(err).Msg("status unmarshal error")
	}
}

func (h *snapcast) handleServerStatusUpdate(status *Status, controller *chik.Controller) {
	h.status.Set(status.Server, controller)
	return
}

func (h *snapcast) handleClientVolumeChange(volume *ClientVolume, controller *chik.Controller) {
	h.status.Edit(controller, func(current interface{}) (interface{}, bool) {
		status := current.(ServerStatus)
		forceChange := false
		for gidx, group := range status.Groups {
			for cidx, client := range group.Clients {
				if client.ID == volume.ID {
					status.Groups[gidx].Clients[cidx].Config.Volume = volume.Volume
					forceChange = true
				}
			}
		}
		return status, forceChange
	})
}

func (h *snapcast) handleGroupStreamChanged(groupStream *GroupStreamChanged, controller *chik.Controller) {
	h.status.Edit(controller, func(current interface{}) (interface{}, bool) {
		status := current.(ServerStatus)
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

func (h *snapcast) HandleChannelEvent(event interface{}, controller *chik.Controller) {
	req, ok := event.(*jrpc2.Request)
	if !ok {
		logger.Error().Msgf("unexpected channel event %v", req)
		return
	}
	logger.Info().Str("method", req.Method()).Msg("Notification received")

	switch req.Method() {
	case "Server.OnUpdate":
		var status Status
		err := req.UnmarshalParams(&status)
		if err != nil {
			logger.Err(err).Msg("notify decode failed")
			return
		}
		h.handleServerStatusUpdate(&status, controller)

	case "Client.OnVolumeChanged":
		var clientVolume ClientVolume
		err := req.UnmarshalParams(&clientVolume)
		if err != nil {
			logger.Err(err).Msg("client volume notification decoding failed")
			return
		}
		h.handleClientVolumeChange(&clientVolume, controller)

	case "Group.OnStreamChanged":
		var groupStream GroupStreamChanged
		err := req.UnmarshalParams(&groupStream)
		if err != nil {
			logger.Err(err).Msg("Failed to decode group stream changed notification")
			return
		}
		h.handleGroupStreamChanged(&groupStream, controller)
	}

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
	h.snapcastRequest("Client.SetVolume", map[string]interface{}{"id": command.ClientID, "volume": volume})
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
			reply, err = h.snapcastRequest("Group.SetStream", map[string]interface{}{"id": command.GroupID, "stream_id": command.Stream})
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
			h.snapcastRequest("Group.SetClients", map[string]interface{}{"id": command.GroupID, "clients": command.Clients})
		}
	case types.GET:
		// TODO: get group description
	}

	return
}

func (h *snapcast) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	logger.Debug().Msg("message received")
	switch message.Command().Type {
	case types.SnapcastClientCommandType:
		h.handleClientCommand(message, controller)
	case types.SnapcastGroupCommandType:
		h.handleGroupCommand(message, controller)
	}
	return nil
}

func (h *snapcast) String() string {
	return "snapcast"
}
