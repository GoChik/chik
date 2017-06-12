package handlers

import (
	"fmt"
	"iosomething"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type forwarding struct {
	iosomething.BaseHandler
	peers map[uuid.UUID]chan<- *iosomething.Message
}

func NewForwardingHandler(peers map[uuid.UUID]chan<- *iosomething.Message) iosomething.Handler {
	return &forwarding{
		BaseHandler: iosomething.NewHandler(""),
		peers:       peers,
	}
}

func (h *forwarding) SetUp(remote chan<- *iosomething.Message) chan bool {
	h.Remote = remote
	return h.Error
}

func (h *forwarding) TearDown() {
	logrus.Debug(fmt.Sprintf("Disconnecting peer %v", h.ID))
	delete(h.peers, h.ID)
}

func (h *forwarding) Handle(message *iosomething.Message) {
	sender, err := message.SenderUUID()
	if err != nil {
		logrus.Error("Unable to get sender UUID", err)
		return
	}

	if h.ID == uuid.Nil && sender != uuid.Nil {
		logrus.Debug(fmt.Sprintf("Adding peer %v", sender))
		h.ID = sender
		h.peers[h.ID] = h.Remote
	}

	receiver, err := message.ReceiverUUID()
	if err != nil {
		logrus.Error("Unable to read receiver UUID", err)
	}

	if receiver == uuid.Nil {
		// No reciver specified
		logrus.Debug("No receiver specified")
		return
	}

	receiverRemote := h.peers[receiver]
	if receiverRemote == nil {
		logrus.Error(fmt.Sprintf("%v disconnected: ", receiver))
		return
	}

	receiverRemote <- message
	logrus.Debug("Message forwarded")
}
