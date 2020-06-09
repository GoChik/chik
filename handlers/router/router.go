package router

import (
	"errors"
	"fmt"
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

func (h *forwarding) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	logger.Debug().Msg("Received a message to route")
	sender := message.SenderUUID()
	if sender == uuid.Nil {
		err := errors.New("Unable to get sender UUID")
		logger.Err(err)
		return err
	}

	if h.id == uuid.Nil {
		logger.Debug().Msgf("Adding peer %v", sender)
		h.id = sender
		h.peers.Store(h.id, controller)
	} else if h.id != sender {
		err := fmt.Errorf("Unexpected sender, expecting: %v got: %v", h.id, sender)
		logger.Err(err)
		return err
	}

	receiver, err := message.ReceiverUUID()
	if err != nil {
		logger.Warn().Msgf("Unable to read receiver UUID: %v", err)
		return nil
	}

	switch receiver {
	case uuid.Nil:
		logger.Warn().Msg("No receiver specified")
		return nil

	case h.id:
	case controller.ID:
		logger.Warn().Msg("Ignoring message to self")
		return nil

	default:
		logger.Info().Msgf("Forwarding a message to: %v", receiver)

		receiverRemote, _ := h.peers.Load(receiver)
		if receiverRemote == nil {
			logger.Error().Msgf("Peer disconnected: %v", receiver)
			return nil
		}

		receiverRemote.(*chik.Controller).PubMessage(message, types.AnyOutgoingCommandType.String())
	}
	return nil
}

func (h *forwarding) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *forwarding) Teardown() {
	logger.Info().Msgf("Disconnecting peer: %v", h.id)
	h.peers.Delete(h.id)
}

func (h *forwarding) String() string {
	return "router"
}
