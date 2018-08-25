package handlers

import (
	"fmt"
	"iosomething"
	"sync"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type forwarding struct {
	id    uuid.UUID
	peers *sync.Map
}

func NewForwardingHandler(peers *sync.Map) iosomething.Handler {
	return &forwarding{
		id:    uuid.Nil,
		peers: peers,
	}
}

func (h *forwarding) terminate() {
	logrus.Debug(fmt.Sprintf("Disconnecting peer %v", h.id))
	h.peers.Delete(h.id)
}

func (h *forwarding) HandlerRoutine(remote *iosomething.Remote) {
	logrus.Debug("starting forwarding handler")

	defer h.terminate()
	defer remote.Terminate()

	in := remote.PubSub.Sub("in")
	for data := range in {
		message := data.(*iosomething.Message)
		sender, err := message.SenderUUID()
		if err != nil {
			logrus.Error("Unable to get sender UUID ", err)
			break
		}

		if sender == uuid.Nil {
			logrus.Error("Empty UUID")
			// TODO: maybe we can trigger an error here (to check if it is possible that we have messages from unknown peers)
			break
		}

		if h.id == uuid.Nil {
			logrus.Debug(fmt.Sprintf("Adding peer %v", sender))
			h.id = sender
			h.peers.Store(h.id, remote)
		} else if h.id != sender {
			logrus.Errorf("Unexpected sender, expecting: %v got: %v", h.id, sender)
			remote.Terminate()
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

			receiverRemote.(*iosomething.Remote).PubSub.Pub(message, "out")
		}
	}
	logrus.Debug("shutting down forwarding handler")
}

func (h *forwarding) Status() interface{} {
	return nil
}

func (h *forwarding) String() string {
	return "forward"
}
