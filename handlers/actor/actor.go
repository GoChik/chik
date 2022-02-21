package actor

import (
	"encoding/json"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	"github.com/thoas/go-funk"
)

// TODO: Perform may also contain an uuid in order to send a remote notification to an app

var logger = log.With().Str("handler", "actor").Logger()

const configKey = "storage.actions"

type ActionCommand struct {
	Action types.Action `json:"action" mapstructure:"action"`
	Value  Action       `json:"value,omitempty" mapstructure:"value"`
}

// Action is composed of a list of Queries and a Command to perform in case the AND composition of queries returns true
type Action struct {
	ID      string           `json:"id"`
	Query   StateQueries     `json:"query,omitempty"`
	Perform []*types.Command `json:"perform,omitempty"`
}

type actor struct {
	chik.BaseHandler
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

	return &actor{
		actions: actions,
	}
}

func (h *actor) executeActions(controller *chik.Controller, currentState map[string]interface{}) {
	state := CreateState(h.previousState, currentState)

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

func (h *actor) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	switch message.Command().Type {
	case types.StatusNotificationCommandType:
		var status types.Status
		json.Unmarshal(message.Command().Data, &status)
		h.executeActions(controller, status)
		h.previousState = status

	case types.ActionRequestCommandType:
		var request ActionCommand
		err := json.Unmarshal(message.Command().Data, &request)
		if err != nil {
			logger.Err(err).Msg("Failed decoding ActionRequest")
			return err
		}
		switch request.Action {
		case types.GET:
			controller.Reply(message, types.ActionReplyCommandType, h.actions)
			// TODO: allow to make query on actions to get

		case types.SET:
			var found bool
			for i, v := range h.actions {
				if v.ID == request.Value.ID {
					logger.Info().Msgf("Action %s found: modifying it", v.ID)
					h.actions[i] = request.Value
					found = true
				}
			}
			if !found {
				logger.Info().Msgf("Action %s not found: creating a new action", request.Value.ID)
				h.actions = append(h.actions, request.Value)
			}
			controller.Reply(message, types.ActionReplyCommandType, h.actions)

		case types.RESET:
			h.actions = funk.Filter(h.actions, func(action Action) bool {
				return action.ID != request.Value.ID
			}).([]Action)
			controller.Reply(message, types.ActionReplyCommandType, h.actions)
		}
	}

	return nil
}

func (h *actor) String() string {
	return "actions"
}
