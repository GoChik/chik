package actor

import (
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

///////// Todos in priority order /////////
// TODO: Perform may also contain an uuid in order to send a remote notification to an app

const configKey = "storage.actions"

// Action is composed of a list of Queries and a Command to perform in case the AND composition of queries returns true
type Action struct {
	ID      string         `json:"id"`
	Query   []StateQuery   `json:"query"`
	Perform *types.Command `json:"perform"`
}

type actor struct {
	actions       map[string]Action
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

	return &actor{funk.ToMap(actions, "ID").(map[string]Action), nil}
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
	return []string{"io", "status", "time", "sun"}
}

func (h *actor) Topics() []types.CommandType {
	return []types.CommandType{types.StatusNotificationCommandType}
}

func (h *actor) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewEmptyTimer()
}

func (h *actor) HandleMessage(message *chik.Message, controller *chik.Controller) {
	var status types.Status
	json.Unmarshal(message.Command().Data, &status)
	h.executeActions(controller, status)
	h.previousState = status
}

func (h *actor) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *actor) Teardown() {}

func (h *actor) String() string {
	return "actions"
}
