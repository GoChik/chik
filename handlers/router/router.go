package router

import (
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type forwarding struct {
	id    uuid.UUID
	peers *sync.Map
}

func New(peers *sync.Map) chik.Handler {
	return &forwarding{
		id:    uuid.Nil,
		peers: peers,
	}
}

func (h *forwarding) Dependencies() []string {
	return []string{}
}

func (h *forwarding) Topics() []types.CommandType {
	return []types.CommandType{types.AnyIncomingCommandType}
}

func (h *forwarding) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
}

func (h *forwarding) HandleMessage(message *chik.Message, controller *chik.Controller) {
	logrus.Debug("Received a message to route")
	sender := message.SenderUUID()
	if sender == uuid.Nil {
		logrus.Error("Unable to get sender UUID")
		controller.Shutdown()
		return
	}

	if h.id == uuid.Nil {
		logrus.Debugf("Adding peer %v", sender)
		h.id = sender
		h.peers.Store(h.id, controller)
	} else if h.id != sender {
		logrus.Errorf("Unexpected sender, expecting: %v got: %v", h.id, sender)
		controller.Shutdown()
		return
	}

	receiver, err := message.ReceiverUUID()
	if err != nil {
		logrus.Warn("Unable to read receiver UUID", err)
		return
	}

	switch receiver {
	case uuid.Nil:
		logrus.Warning("No receiver specified")
		return

	case h.id:
	case controller.ID:
		logrus.Warning("Ignoring message to self")
		return

	default:
		logrus.Debug("Forwarding a message to: ", receiver)

		receiverRemote, _ := h.peers.Load(receiver)
		if receiverRemote == nil {
			logrus.Errorf("%v disconnected: ", receiver)
			return
		}

		receiverRemote.(*chik.Controller).PubMessage(message, types.AnyOutgoingCommandType.String())
	}
}

func (h *forwarding) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *forwarding) Teardown() {
	logrus.Debugf("Disconnecting peer %v", h.id)
	h.peers.Delete(h.id)
}

func (h *forwarding) String() string {
	return "router"
}
