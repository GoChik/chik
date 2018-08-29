package handlers

import (
	"iosomething"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/gofrs/uuid"
)

const maxErrors uint32 = 3

type heartbeat struct {
	id       uuid.UUID
	interval time.Duration
	errors   uint32
}

// NewHeartBeatHandler creates a new heartbeat handler
func NewHeartBeatHandler(identity uuid.UUID, interval time.Duration) iosomething.Handler {
	if interval <= 100*time.Millisecond {
		logrus.Error("Interval value too low: ", interval)
		return nil
	}
	return &heartbeat{
		id:       identity,
		interval: interval,
	}
}

func (h *heartbeat) Run(remote *iosomething.Remote) {
	sendHeartBeat := func() {
		logrus.Debug("Sending heartbeat")
		remote.PubSub.Pub(iosomething.NewMessage(iosomething.HeartbeatType, h.id, uuid.Nil, []byte{}), "out")
	}

	logrus.Debug("starting heartbeat handler")
	sendHeartBeat()

	in := remote.PubSub.Sub(iosomething.HeartbeatType.String())
	for {
		select {
		case data, more := <-in:
			if !more {
				logrus.Debug("Shutting down heartbeat")
				return
			}
			message := data.(*iosomething.Message)

			if message.Type() == iosomething.HeartbeatType {
				h.errors = 0
			}

		case <-time.After(h.interval):
			sendHeartBeat()
			h.errors++
			if h.errors >= maxErrors {
				logrus.Error("Heartbeat error threshold exceeded: shutting down remote connection")
				remote.Terminate()
				return
			}
		}
	}
}

func (h *heartbeat) Status() interface{} {
	return nil
}

func (h *heartbeat) String() string {
	return "heartbeat"
}
