package status

import (
	"encoding/json"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type set struct{}

type handler struct {
	subscribers   map[uuid.UUID]set
	currentStatus types.Status
}

func New() chik.Handler {
	return &handler{
		subscribers:   make(map[uuid.UUID]set, 0),
		currentStatus: types.Status{},
	}
}

func (h *handler) Run(remote *chik.Controller) {
	incoming := remote.Sub(types.StatusSubscriptionCommandType.String(), types.StatusUpdateCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		logrus.Debug("status: received: ", message)

		if message.Command().Type == types.StatusSubscriptionCommandType {
			if message.SenderUUID() != chik.LoopbackID ||
				message.SenderUUID() != remote.ID {
				h.subscribers[message.SenderUUID()] = set{}
			}
			remote.Reply(message, types.StatusNotificationCommandType, h.currentStatus)
			continue
		}

		if message.Command().Type == types.StatusUpdateCommandType {
			var status types.Status
			err := json.Unmarshal(message.Command().Data, &status)
			if err != nil {
				logrus.Warning("Failed to decode Status update command: ", err)
				continue
			}
			for k, v := range status {
				h.currentStatus[k] = v
			}
			for id := range h.subscribers {
				remote.Pub(types.NewCommand(types.StatusNotificationCommandType, h.currentStatus), id)
			}
			continue
		}

		logrus.Warning("Unexpected message in status handler ", message)
	}
}

func (h *handler) String() string {
	return "status"
}