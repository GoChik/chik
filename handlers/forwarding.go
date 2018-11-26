package handlers

import (
	"chik"
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"
	uuid "github.com/gofrs/uuid"
)

type forwarding struct {
	id    uuid.UUID
	peers *sync.Map
}

func NewForwardingHandler(peers *sync.Map) chik.Handler {
	return &forwarding{
		id:    uuid.Nil,
		peers: peers,
	}
}

func (h *forwarding) terminate() {
	logrus.Debug(fmt.Sprintf("Disconnecting peer %v", h.id))
	h.peers.Delete(h.id)
}

func (h *forwarding) Run(controller *chik.Controller) {
	logrus.Debug("starting forwarding handler")

	defer h.terminate()

	in := controller.PubSub.Sub("in")
	for data := range in {
		message := data.(*chik.Message)
		sender := message.SenderUUID()
		if sender == uuid.Nil {
			logrus.Error("Unable to get sender UUID")
			break
		}

		if h.id == uuid.Nil {
			logrus.Debug(fmt.Sprintf("Adding peer %v", sender))
			h.id = sender
			h.peers.Store(h.id, controller)
		} else if h.id != sender {
			logrus.Errorf("Unexpected sender, expecting: %v got: %v", h.id, sender)
			controller.Shutdown()
			break
		}

		receiver, err := message.ReceiverUUID()
		if err != nil {
			logrus.Warn("Unable to read receiver UUID", err)
			continue
		}

		switch receiver {
		case uuid.Nil:
			logrus.Warning("No receiver specified")
			continue

		case h.id:
			logrus.Warning("Not forwarding message to self")
			continue

		default:
			logrus.Debug("Forwarding a message to: ", receiver)

			receiverRemote, _ := h.peers.Load(receiver)
			if receiverRemote == nil {
				logrus.Error(fmt.Sprintf("%v disconnected: ", receiver))
				continue
			}

			receiverRemote.(*chik.Controller).PubSub.Pub(message, "out")
		}
	}
	h.peers.Delete(h.id)
	logrus.Debug("shutting down forwarding handler")
}

func (h *forwarding) String() string {
	return "forward"
}
