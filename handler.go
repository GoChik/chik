package iosomething

import (
	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

// Handler is the interface that handles network messages
// and optionally can return a reply
type Handler interface {
	SetUp(outChannel chan<- *Message)
	TearDown()
	Handle(message *Message)
}

type BaseHandler struct {
	ID     uuid.UUID
	Remote chan<- *Message
}

func NewHandler(identity string) BaseHandler {
	id, err := uuid.FromString(identity)
	if err != nil || id == uuid.Nil {
		logrus.Warn("Identity error: ", err)
	}
	return BaseHandler{id, nil}
}
