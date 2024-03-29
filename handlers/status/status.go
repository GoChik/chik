package status

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "status").Logger()

type set struct{}

type subscrier struct {
	listeners  map[string]set
	lastConact time.Time
}

type handler struct {
	chik.BaseHandler
	subscribers   map[uuid.UUID]subscrier
	currentStatus types.Status
}

type StatusCommand struct {
	Action types.Action `json:",int"`
	Query  string       `json:",omitempty"`
	Value  interface{}  `json:",omitempty"`
}

// New creates a new status handler
func New() chik.Handler {
	return &handler{
		subscribers:   make(map[uuid.UUID]subscrier, 0),
		currentStatus: types.Status{},
	}
}

func (h *handler) Dependencies() []string {
	return []string{}
}

func (h *handler) Topics() []types.CommandType {
	return []types.CommandType{
		types.StatusCommandType,
		types.StatusUpdateCommandType,
	}
}

func (h *handler) getStatus(query string) types.Status {
	if query == "" {
		return h.currentStatus
	}
	val, found := h.currentStatus[query]
	if !found {
		return types.Status{}
	}
	return map[string]interface{}{query: val}
}

func (h *handler) HandleMessage(message *chik.Message, remote *chik.Controller) error {
	switch message.Command().Type {
	case types.StatusCommandType:
		var content StatusCommand
		err := json.Unmarshal(message.Command().Data, &content)
		if err != nil {
			logger.Error().Msgf("Failed to decode status subscription command: %v", err)
			return nil
		}

		if content.Action == types.SET &&
			message.SenderUUID() != chik.LoopbackID &&
			message.SenderUUID() != remote.ID {
			logger.Info().Msgf("Subscribing %v to %v", message.SenderUUID(), content.Query)
			current := h.subscribers[message.SenderUUID()]
			if current.listeners == nil {
				current.listeners = make(map[string]set, 1)
			}
			current.listeners[content.Query] = set{}
			current.lastConact = time.Now()
			h.subscribers[message.SenderUUID()] = current
		}

		remote.Reply(message, types.StatusNotificationCommandType, h.getStatus(content.Query))

	case types.StatusUpdateCommandType:
		logger.Debug().Msgf("Status update received: %v", message)
		var status types.Status
		err := json.Unmarshal(message.Command().Data, &status)
		if err != nil {
			logger.Warn().Msgf("Failed to decode Status update: %v", err)
			return nil
		}

		// here there is just one iteration because status is a pair[string]interface{}
		inactiveSubscribers := []uuid.UUID{}
		for k, v := range status {
			h.currentStatus[k] = v
			for id, data := range h.subscribers {
				if time.Now().Sub(data.lastConact) > 10*time.Minute {
					inactiveSubscribers = append(inactiveSubscribers, id)
				}
				_, exists := data.listeners[k]
				if exists {
					remote.Pub(types.NewCommand(types.StatusNotificationCommandType, map[string]interface{}{k: v}), id)
				}
			}
		}

		for _, s := range inactiveSubscribers {
			delete(h.subscribers, s)
		}

		remote.Pub(types.NewCommand(types.StatusNotificationCommandType, h.currentStatus), chik.LoopbackID)
	}
	return nil
}

func (h *handler) String() string {
	return "status"
}
