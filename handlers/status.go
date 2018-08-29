package handlers

import (
	"encoding/json"
	"iosomething"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

type handler struct {
	id       uuid.UUID
	handlers []iosomething.Handler
}

func NewStatusHandler(id uuid.UUID, handlers []iosomething.Handler) iosomething.Handler {
	return &handler{id, handlers}
}

func (h *handler) Run(remote *iosomething.Remote) {
	incoming := remote.PubSub.Sub(iosomething.SimpleCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*iosomething.Message)
		command := iosomething.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil || command.Command != iosomething.GET_STATUS {
			continue
		}

		status := make(iosomething.Status)
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

		reply := iosomething.NewMessage(iosomething.StatusIndicationType, h.id, sender, replyData)
		remote.PubSub.Pub(reply, "out")
	}
}

func (h *handler) Status() interface{} {
	return nil
}

func (h *handler) String() string {
	return "status"
}
