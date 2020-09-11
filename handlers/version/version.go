package version

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "update").Logger()

type version struct {
	Current string
}

// New creates a version holder
func New(currentVersion string) chik.Handler {
	logger.Debug().Msgf("Version: %v", currentVersion)

	return &version{currentVersion}
}

func (h *version) Dependencies() []string {
	return []string{}
}

func (h *version) Topics() []types.CommandType {
	return []types.CommandType{types.VersionRequestCommandType}
}

func (h *version) Setup(controller *chik.Controller) chik.Timer {
	// TODO: check periodically for new versions
	return chik.NewEmptyTimer()
}

func (h *version) HandleMessage(message *chik.Message, remote *chik.Controller) error {
	var command types.SimpleCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Warn().Msg("Unexpected message")
		return nil
	}

	switch command.Action {
	case types.GET:
		logger.Debug().Msgf("Sendingcurrent software version: %v", h.Current)
		version := types.VersionIndication{h.Current, h.Current}
		remote.Reply(message, types.VersionReplyCommandType, version)
	}
	return nil
}

func (h *version) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *version) Teardown() {}

func (h *version) String() string {
	return "version"
}
