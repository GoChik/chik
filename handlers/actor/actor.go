package actor

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

// TODO: Perform may also contain an uuid in order to send a remote notification to an app

var logger = log.With().Str("handler", "actor").Logger()

const configKey = "storage.actions"

// Action is composed of a list of Queries and a Command to perform in case the AND composition of queries returns true
type Action struct {
	ID      string           `json:"id"`
	Query   []StateQuery     `json:"query"`
	Perform []*types.Command `json:"perform"`
}

type actor struct {
	actions       []Action
	previousState map[string]interface{}
}

// New creates a new actor handler
func New() chik.Handler {
	actions := make([]Action, 0)
	err := config.GetStruct(configKey, &actions, StringInterfaceToStateQuery)
	if err != nil {
		logger.Warn().Msgf("Cannot get actions form config file: %v", err)
		config.Set(configKey, actions)
	}

	return &actor{actions, nil}
}

func (h *actor) executeActions(controller *chik.Controller, currentState map[string]interface{}) {
	state := &State{
		Previous: h.previousState,
		Current:  currentState,
	}

	for _, action := range h.actions {
		composedResult := QueryResult{true, false}
		for _, query := range action.Query {
			result, err := query.Execute(state)
			if err != nil {
				logger.Warn().Msgf("State query failed: %v", err)
			}
			composedResult = QueryResult{
				composedResult.match && result.match,
				composedResult.changedSincePreviousEvaluation || result.changedSincePreviousEvaluation,
			}
			// Break early if result does not match
			if !composedResult.match {
				break
			}
		}

		if composedResult.match && composedResult.changedSincePreviousEvaluation {
			logger.Info().Msgf("Query returned positive results, executing: %v", action.Perform)
			for _, command := range action.Perform {
				controller.Pub(command, chik.LoopbackID)
			}
		}
	}
}

func (h *actor) Dependencies() []string {
	return []string{"io", "status", "time"}
}

func (h *actor) Topics() []types.CommandType {
	return []types.CommandType{
		types.StatusNotificationCommandType,
		types.ActionRequestCommandType,
	}
}

func (h *actor) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
}

func (h *actor) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	switch message.Command().Type {
	case types.StatusNotificationCommandType:
		var status types.Status
		json.Unmarshal(message.Command().Data, &status)
		h.executeActions(controller, status)
		h.previousState = status

	case types.ActionRequestCommandType:
		// TODO: allow to make query on actions to get
		controller.Reply(message, types.ActionReplyCommandType, h.actions)
	}

	return nil
}

func (h *actor) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *actor) Teardown() {}

func (h *actor) String() string {
	return "actions"
}
