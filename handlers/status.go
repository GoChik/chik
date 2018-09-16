package handlers

import (
	"encoding/json"
	"chik"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

type handler struct {
	id       uuid.UUID
	handlers []chik.Handler
}

func NewStatusHandler(id uuid.UUID, handlers []chik.Handler) chik.Handler {
	return &handler{id, handlers}
}

func (h *handler) Run(remote *chik.Remote) {
	incoming := remote.PubSub.Sub(chik.SimpleCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		command := chik.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil || command.Command != chik.GET_STATUS {
			continue
		}

		status := make(chik.Status)
		// compose version message
		for _, handler := range h.handlers {
			status[handler.String()] = handler.Status()
		}

		replyData, err := json.Marshal(status)
		if err != nil {
			logrus.Error("Cannot marshal status message")
			continue
		}
		sender, err := message.SenderUUID()
		if err != nil {
			logrus.Error("Cannot determine status request sender")
			continue
		}

		reply := chik.NewMessage(chik.StatusIndicationType, h.id, sender, replyData)
		remote.PubSub.Pub(reply, "out")
	}
}

func (h *handler) Status() interface{} {
	return nil
}

func (h *handler) String() string {
	return "status"
}
