package handlers

import (
	"chik"
	"encoding/json"
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
		if err != nil || len(command.Command) != 1 || command.Command[0] != chik.GET {
			continue
		}

		status := make(chik.Status)
		// compose version message
		for _, handler := range h.handlers {
			status[handler.String()] = handler.Status()
		}

		remote.Reply(message, chik.StatusReplyCommandType, status)
	}
}

func (h *handler) Status() interface{} {
	return nil
}

func (h *handler) String() string {
	return "status"
}
