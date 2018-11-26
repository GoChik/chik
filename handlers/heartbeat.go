package handlers

import (
	"chik"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/gofrs/uuid"
)

const maxErrors uint32 = 3

type heartbeat struct {
	interval time.Duration
	errors   uint32
}

// NewHeartBeatHandler creates a new heartbeat handler
func NewHeartBeatHandler(interval time.Duration) chik.Handler {
	if interval <= 100*time.Millisecond {
		logrus.Error("Interval value too low: ", interval)
		return nil
	}
	return &heartbeat{
		interval: interval,
	}
}

func (h *heartbeat) sender(controller *chik.Controller) *time.Ticker {
	sendHeartBeat := func() {
		logrus.Debug("Sending heartbeat")
		controller.PubSub.Pub(chik.NewMessage(uuid.Nil, chik.NewCommand(chik.HeartbeatType, nil)), "out")
	}

	ticker := time.NewTicker(h.interval)
	go func() {
		sendHeartBeat()
		for range ticker.C {
			sendHeartBeat()
			atomic.AddUint32(&h.errors, 1)
			if h.errors >= maxErrors {
				logrus.Error("Heartbeat error threshold exceeded: shutting down remote connection")
				controller.Disconnect()
				return
			}
		}
	}()
	return ticker
}

func (h *heartbeat) Run(controller *chik.Controller) {
	logrus.Debug("starting heartbeat handler")
	time.Sleep(1 * time.Second)
	sender := h.sender(controller)
	defer sender.Stop()

	in := controller.PubSub.Sub(chik.HeartbeatType.String())
	for data := range in {
		message := data.(*chik.Message)

		if message.Command().Type == chik.HeartbeatType {
			atomic.StoreUint32(&h.errors, 0)
		}
	}
	logrus.Debug("Shutting down heartbeat")
}

func (h *heartbeat) String() string {
	return "heartbeat"
}
