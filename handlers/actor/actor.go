package actor

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
)

///////// Todos in priority order /////////
// TODO: Perform may also contain an uuid in order to send a remote notification to an app
// TODO: There might be an operation to add an action to the actor from a mobile application (in order to replace timers)

const configKey = "storage.actions"

// Action is composed of a list of Queries and a Command to perform in case the AND composition of queries returns true
type Action struct {
	Query   []StateQuery
	Perform *types.Command
}

type actor struct {
	actions       []Action
	previousState interface{}
}

// New creates a new actor handler
func New() chik.Handler {
	actions := make([]Action, 0)
	err := config.GetStruct(configKey, &actions, StringInterfaceToStateQuery)
	if err != nil {
		logrus.Warn("Cannot get actions form config file: ", err)
		config.Set(configKey, actions)
	}
	return &actor{actions, nil}
}

func (h *actor) executeActions(controller *chik.Controller, currentState interface{}) {
	state := &State{
		Previous: h.previousState,
		Current:  currentState,
	}

	for _, action := range h.actions {
		composedResult := true
		for _, query := range action.Query {
			result, err := query.Execute(state)
			if err != nil {
				logrus.Warn("State query failed: ", err)
			}
			composedResult = (result && composedResult)
		}

		if composedResult {
			logrus.Debug("Query returned positive results, executing: ", action.Perform)
			controller.Pub(action.Perform, chik.LoopbackID)
		}
	}
}

func (h *actor) Dependencies() []string {
	return []string{"io", "status"}
}

func (h *actor) Topics() []types.CommandType {
	return []types.CommandType{types.StatusUpdateCommandType}
}

func (h *actor) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
}

func (h *actor) HandleMessage(message *chik.Message, controller *chik.Controller) {
	var status types.Status
	json.Unmarshal(message.Command().Data, &status)
	currentState := status["io"]
	h.executeActions(controller, currentState)
	h.previousState = currentState
}

func (h *actor) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *actor) Teardown() {}

func (h *actor) String() string {
	return "actions"
}
