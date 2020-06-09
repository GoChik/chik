package router

import (
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	uuid "github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "router").Logger()

type forwarding struct {
	id    uuid.UUID
	peers *sync.Map
}

func New(peers *sync.Map) chik.Handler {
	return &forwarding{
		id:    uuid.Nil,
		peers: peers,
	}
}

func (h *forwarding) Dependencies() []string {
	return []string{}
}

func (h *forwarding) Topics() []types.CommandType {
	return []types.CommandType{types.AnyIncomingCommandType}
}

func (h *forwarding) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
}

func (h *forwarding) HandleMessage(message *chik.Message, controller *chik.Controller) {
	logger.Debug().Msg("Received a message to route")
	sender := message.SenderUUID()
	if sender == uuid.Nil {
		logger.Error().Msg("Unable to get sender UUID")
		controller.Shutdown()
		return
	}

	if h.id == uuid.Nil {
		logger.Debug().Msgf("Adding peer %v", sender)
		h.id = sender
		h.peers.Store(h.id, controller)
	} else if h.id != sender {
		logger.Error().Msgf("Unexpected sender, expecting: %v got: %v", h.id, sender)
		controller.Shutdown()
		return
	}

	receiver, err := message.ReceiverUUID()
	if err != nil {
		logger.Warn().Msgf("Unable to read receiver UUID: %v", err)
		return
	}

	switch receiver {
	case uuid.Nil:
		logger.Warn().Msg("No receiver specified")
		return

	case h.id:
	case controller.ID:
		logger.Warn().Msg("Ignoring message to self")
		return

	default:
		logger.Info().Msgf("Forwarding a message to: %v", receiver)

		receiverRemote, _ := h.peers.Load(receiver)
		if receiverRemote == nil {
			logger.Error().Msgf("Peer disconnected: %v", receiver)
			return
		}

		receiverRemote.(*chik.Controller).PubMessage(message, types.AnyOutgoingCommandType.String())
	}
}

func (h *forwarding) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *forwarding) Teardown() {
	logger.Info().Msgf("Disconnecting peer: %v", h.id)
	h.peers.Delete(h.id)
}

func (h *forwarding) String() string {
	return "router"
}
