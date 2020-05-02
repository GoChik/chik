package status

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

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

func (h *handler) HandleMessage(message *chik.Message, remote *chik.Controller) {
	switch message.Command().Type {
	case types.StatusSubscriptionCommandType:
		var content SubscriptionCommand
		err := json.Unmarshal(message.Command().Data, &content)
		if err == nil && content.Command == types.SET {
			if message.SenderUUID() != chik.LoopbackID &&
				message.SenderUUID() != remote.ID {
				logrus.Debug("Registering subscriber ", message.SenderUUID(), content.Query)
				current := h.subscribers[message.SenderUUID()]
				if current.listeners == nil {
					current.listeners = make(map[string]set, 1)
				}
				current.listeners[content.Query] = set{}
				current.lastConact = time.Now()
				h.subscribers[message.SenderUUID()] = current
			}
		}
		remote.Reply(message, types.StatusNotificationCommandType, h.currentStatus)

	case types.StatusUpdateCommandType:
		logrus.Debug("Status update received ", message)
		var status types.Status
		err := json.Unmarshal(message.Command().Data, &status)
		if err != nil {
			logrus.Warning("Failed to decode Status update command: ", err)
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
	}
}

func (h *handler) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *handler) Teardown() {}

func (h *handler) String() string {
	return "status"
}
