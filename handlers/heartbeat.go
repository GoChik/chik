package handlers

import (
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
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

func (h *heartbeat) sender(controller *chik.Controller, receiverID uuid.UUID) *time.Ticker {
	sendHeartBeat := func() {
		logrus.Debug("Sending heartbeat")
		controller.Pub(types.NewCommand(types.HeartbeatType, nil), receiverID)
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
	var senderRoutine *time.Ticker

	in := controller.Sub(types.HeartbeatType.String())
	for data := range in {
		message := data.(*chik.Message)
		if senderRoutine == nil {
			senderRoutine = h.sender(controller, message.SenderUUID())
		}

		if message.Command().Type == types.HeartbeatType {
			atomic.StoreUint32(&h.errors, 0)
		}
	}
	if senderRoutine != nil {
		senderRoutine.Stop()
	}
	logrus.Debug("Shutting down heartbeat")
}

func (h *heartbeat) String() string {
	return "heartbeat"
}
