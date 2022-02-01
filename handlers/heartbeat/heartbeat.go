package heartbeat

import (
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "heartbeat").Logger()

const maxErrors uint32 = 300

type heartbeat struct {
	chik.BaseHandler
	errors   uint32
	remoteID uuid.UUID
}

// New creates a new heartbeat handler
func New() chik.Handler {
	return &heartbeat{}
}

func (h *heartbeat) Topics() []types.CommandType {
	return []types.CommandType{types.HeartbeatType}
}

func (h *heartbeat) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	return chik.Interrupts{Timer: chik.NewTimer((chik.MaxIdleTime/3)*2, true)}, nil
}

func (h *heartbeat) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	if h.remoteID == uuid.Nil {
		h.remoteID = message.SenderUUID()
	}

	if message.Command().Type == types.HeartbeatType {
		logger.Debug().Msg("Heartbeat received")
		h.errors = 0
	}
	return nil
}

func (h *heartbeat) HandleTimerEvent(tick time.Time, controller *chik.Controller) error {
	logger.Debug().Msg("Sending heartbeat")
	controller.PubMessage(chik.NewMessage(h.remoteID, types.NewCommand(types.HeartbeatType, nil)),
		types.AnyOutgoingCommandType.String())
	h.errors = h.errors + 1
	if h.errors >= maxErrors {
		logger.Error().Msg("Heartbeat threshold exceeded: shutting down remote connection")
		controller.Pub(types.NewCommand(types.RemoteStopCommandType, nil), h.remoteID)
	}
	return nil
}

func (h *heartbeat) String() string {
	return "heartbeat"
}
