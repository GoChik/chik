package heartbeat

import (
	"sync/atomic"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "actor").Logger()

const maxErrors uint32 = 300

type heartbeat struct {
	interval time.Duration
	errors   uint32
	remoteID uuid.UUID
}

// New creates a new heartbeat handler
func New(interval time.Duration) chik.Handler {
	if interval <= 100*time.Millisecond {
		logger.Error().Msgf("Interval value too low: %v", interval)
		return nil
	}
	return &heartbeat{
		interval: interval,
	}
}

func (h *heartbeat) Dependencies() []string {
	return make([]string, 0)
}

func (h *heartbeat) Topics() []types.CommandType {
	return []types.CommandType{types.HeartbeatType}
}

func (h *heartbeat) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewTimer(h.interval, true)
}

func (h *heartbeat) HandleMessage(message *chik.Message, controller *chik.Controller) {
	if h.remoteID == uuid.Nil {
		h.remoteID = message.SenderUUID()
	}

	if message.Command().Type == types.HeartbeatType {
		atomic.StoreUint32(&h.errors, 0)
	}
}

func (h *heartbeat) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	logger.Debug().Msg("Sending heartbeat")
	controller.PubMessage(chik.NewMessage(h.remoteID, types.NewCommand(types.HeartbeatType, nil)),
		types.AnyOutgoingCommandType.String())
	atomic.AddUint32(&h.errors, 1)
	if h.errors >= maxErrors {
		logger.Error().Msg("Heartbeat threshold exceeded: shutting down remote connection")
		controller.Disconnect()
		return
	}
}

func (h *heartbeat) Teardown() {}

func (h *heartbeat) String() string {
	return "heartbeat"
}
