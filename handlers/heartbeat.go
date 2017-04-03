package handlers

import (
	"iosomething"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type heartbeat struct {
	iosomething.BaseHandler
	interval time.Duration
	active   bool
	stop     chan bool
}

// NewHeartBeatHandler creates a new heartbeat handler, active is used to specify if the handler is also
// sending heartbeat messages
func NewHeartBeatHandler(identity string, interval time.Duration, active bool) iosomething.Handler {
	return &heartbeat{
		BaseHandler: iosomething.NewHandler(identity),
		interval:    interval,
		active:      active,
		stop:        make(chan bool, 1),
	}
}

func (h *heartbeat) SetUp(remote chan<- *iosomething.Message) {
	h.Remote = remote
	h.Remote <- iosomething.NewMessage(iosomething.MESSAGE, h.ID, uuid.Nil, []byte{})
	if !h.active {
		return
	}

	go func() {
		for {
			select {
			case <-h.stop:
				logrus.Debug("Stopping heartbeat service")
				close(h.stop)
				return

			case <-time.After(h.interval):
				h.Remote <- iosomething.NewMessage(iosomething.HEARTBEAT, h.ID, uuid.Nil, []byte{})
				break
			}
		}
	}()
}

func (h *heartbeat) TearDown() {
	h.stop <- true
}

func (h *heartbeat) Handle(message *iosomething.Message) {
	if message.Type() == iosomething.HEARTBEAT {
		logrus.Debug("Heartbeat received")
	}
}
