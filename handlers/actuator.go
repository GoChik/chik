package handlers

import (
	"iosomething"
	"iosomething/actuator"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type actuatorHandler struct {
	iosomething.BaseHandler
	actuator actuator.Actuator
}

type replyData struct {
	reply  []byte
	sender uuid.UUID
}

// NewActuatorHandler creates an handler that is working with the selected actuator
func NewActuatorHandler(identity string) iosomething.Handler {
	return &actuatorHandler{iosomething.NewHandler(identity), actuator.NewActuator()}
}

func (h *actuatorHandler) SetUp(remote chan<- *iosomething.Message) {
	h.Remote = remote
	h.actuator.Initialize()
}

func (h *actuatorHandler) TearDown() {
	h.actuator.Deinitialize()
}

func (h *actuatorHandler) Handle(message *iosomething.Message) {
	receiver, err := message.ReceiverUUID()
	if message.Type() == iosomething.HEARTBEAT {
		logrus.Debug("Heartbeat received")
		return
	}

	if err != nil || receiver != h.ID {
		logrus.Warn("Message has been dispatched to the wrong receiver")
		return
	}

	reply := h.actuator.Execute(message.Data())
	if len(reply) == 0 {
		return
	}
	sender, err := message.SenderUUID()
	if err != nil {
		logrus.Warning("Reply for no one: ", err)
		return
	}

	h.Remote <- iosomething.NewMessage(iosomething.MESSAGE, h.ID, sender, reply)
}
