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
	subscribers   map[uuid.UUID]subscrier
	currentStatus types.Status
}

type SubscriptionCommand struct {
	Command types.Action `json:",int"`
	Query   string       `json:",omitempty"`
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
		types.StatusSubscriptionCommandType,
		types.StatusUpdateCommandType,
	}
}

func (h *handler) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
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

func (h *handler) HandleMessage(message *chik.Message, remote *chik.Controller) {
	switch message.Command().Type {
	case types.StatusSubscriptionCommandType:
		var content SubscriptionCommand
		err := json.Unmarshal(message.Command().Data, &content)
		if err != nil {
			logger.Error().Msgf("Failed to decode status subscription command: %v", err)
			return
		}

		if content.Command == types.SET &&
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
			return
		}

		// here there is just one iteration because status is a pair[string]interface{}
		for k, v := range status {
			h.currentStatus[k] = v
			for id, data := range h.subscribers {
				_, exists := data.listeners[k]
				if exists {
					remote.Pub(types.NewCommand(types.StatusNotificationCommandType, map[string]interface{}{k: v}), id)
				}
			}
		}

		remote.Pub(types.NewCommand(types.StatusNotificationCommandType, h.currentStatus), chik.LoopbackID)
	}
}

func (h *handler) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *handler) Teardown() {}

func (h *handler) String() string {
	return "status"
}
