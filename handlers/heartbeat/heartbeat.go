package heartbeat

import (
	"sync/atomic"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const maxErrors uint32 = 3

type heartbeat struct {
	interval time.Duration
	errors   uint32
	remoteID uuid.UUID
}

// New creates a new heartbeat handler
func New(interval time.Duration) chik.Handler {
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
		controller.PubMessage(chik.NewMessage(h.remoteID, types.NewCommand(types.HeartbeatType, nil)), chik.OutgoingMessage)
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
	senderRoutine := h.sender(controller)
	in := controller.Sub(types.HeartbeatType.String())
	for data := range in {
		message := data.(*chik.Message)
		if h.remoteID == uuid.Nil {
			h.remoteID = message.SenderUUID()
		}

		if message.Command().Type == types.HeartbeatType {
			atomic.StoreUint32(&h.errors, 0)
		}
	}
	senderRoutine.Stop()
	logrus.Debug("Shutting down heartbeat")
}

func (h *heartbeat) String() string {
	return "heartbeat"
}
