package handlers

import (
	"chik"
	"encoding/json"

	"github.com/gofrs/uuid"

	"github.com/Sirupsen/logrus"
)

type handler struct {
	handlers []chik.Handler
}

func NewStatusHandler(handlers []chik.Handler) chik.Handler {
	return &handler{handlers}
}

func (h *handler) Run(remote *chik.Remote) {
	incoming := remote.PubSub.Sub(chik.StatusRequestCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		command := chik.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil || len(command.Command) == 1 || command.Command[0] != chik.GET {
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

		reply := chik.NewMessage(chik.StatusReplyCommandType, uuid.Nil, sender, replyData)
		remote.PubSub.Pub(reply, "out")
	}
}

func (h *handler) Status() interface{} {
	return nil
}

func (h *handler) String() string {
	return "status"
}
