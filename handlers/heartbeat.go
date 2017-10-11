package handlers

import (
	"iosomething"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

const maxErrors = uint32(3)

type heartbeat struct {
	iosomething.BaseHandler
	interval time.Duration
	stop     chan bool
	errors   uint32
}

// NewHeartBeatHandler creates a new heartbeat handler
func NewHeartBeatHandler(identity string, interval time.Duration) iosomething.Handler {
	return &heartbeat{
		BaseHandler: iosomething.NewHandler(identity),
		interval:    interval,
		stop:        make(chan bool, 1),
	}
}

func (h *heartbeat) SetUp(remote chan<- *iosomething.Message) chan bool {
	h.Remote = remote
	h.Remote <- iosomething.NewMessage(iosomething.MESSAGE, h.ID, uuid.Nil, []byte{})

	go func() {
		for {
			select {
			case <-h.stop:
				logrus.Debug("Stopping heartbeat service")
				return

			case <-time.After(h.interval):
				logrus.Debug("Sending heartbeat")
				h.Remote <- iosomething.NewMessage(iosomething.HEARTBEAT, h.ID, uuid.Nil, []byte{})
				atomic.AddUint32(&h.errors, 1)
				if atomic.LoadUint32(&h.errors) >= maxErrors {
					h.Error <- true
				}
				break
			}
		}
	}()

	return h.Error
}

func (h *heartbeat) TearDown() {
	h.stop <- true
}

func (h *heartbeat) Handle(message *iosomething.Message) {
	if message.Type() == iosomething.HEARTBEAT {
		logrus.Debug("Heartbeat received")
		atomic.StoreUint32(&h.errors, 0)
	}
}
