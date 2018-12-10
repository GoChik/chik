package handlers

import (
	"chik"
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

type set struct{}

type handler struct {
	subscribers   map[uuid.UUID]set
	currentStatus chik.Status
}

func NewStatusHandler() chik.Handler {
	return &handler{
		subscribers:   make(map[uuid.UUID]set, 0),
		currentStatus: chik.Status{},
	}
}

func (h *handler) Run(remote *chik.Controller) {
	incoming := remote.PubSub.Sub(chik.StatusSubscriptionCommandType.String(), chik.StatusUpdateCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)

		if message.Command().Type == chik.StatusSubscriptionCommandType {
			h.subscribers[message.SenderUUID()] = set{}
			remote.Reply(message, chik.StatusNotificationCommandType, h.currentStatus)
			continue
		}

		if message.Command().Type == chik.StatusUpdateCommandType {
			var status chik.Status
			err := json.Unmarshal(message.Command().Data, &status)
			if err != nil {
				logrus.Warning("Failed to decode Status update command: ", err)
				continue
			}
			for k, v := range status {
				h.currentStatus[k] = v
			}
			for id := range h.subscribers {
				remote.PubSub.Pub(chik.NewMessage(id, chik.NewCommand(chik.StatusNotificationCommandType, h.currentStatus)), "out")
			}
			continue
		}

		logrus.Warning("Unexpected message in status handler")
	}
}

func (h *handler) String() string {
	return "status"
}
