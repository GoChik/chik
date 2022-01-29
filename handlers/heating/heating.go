package heating

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "heating").Logger()

const MinimumRunningTime = time.Hour * 2

type room struct {
	ID                   string `mapstructure:"id"`
	CurrentTemperatureID string `mapstructure:"current_temperature_id"`
	TargetTemperatureID  string `mapstructure:"target_temperature_id"`
	ThermalValveID       string `mapstructure:"thermal_valve_id"`
}

type roomStatus struct {
	room                    *room
	currentTemperature      float64
	targetTemperature       float64
	isHeating               bool
	lastHeatingStatusChange time.Time
}

type heating struct {
	chik.BaseHandler
	Rooms     []*room `mapstructure:"rooms"`
	Threshold float64 `json:"threshold"`
}

// New creates an heating controller
func New() chik.Handler {
	h := heating{
		Rooms: make([]*room, 0),
	}

	err := config.GetStruct("heating", &h)
	if err != nil {
		logger.Err(err).Msg("failed parsing conf")
	}
	logger.Debug().Msgf("Heating: %v", h)
	return &h
}

func (h *heating) Dependencies() []string {
	return []string{"io"}
}

func (h *heating) Topics() []types.CommandType {
	return []types.CommandType{types.StatusNotificationCommandType}
}

func (h *heating) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	for _, r := range h.Rooms {
		command := types.DigitalCommand{
			Action:      types.RESET,
			ApplianceID: r.ThermalValveID,
		}
		controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
	}
	return chik.Interrupts{Timer: chik.NewEmptyTimer()}, nil
}

func getValue(status types.Status, id string, key string) (interface{}, error) {
	v, ok := status["io"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Cannot find io in status")
	}
	v, ok = v[id].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Cannot find thermal element %v in status", id)
	}
	return v[key], nil
}

func (h *heating) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	var status types.Status
	err := json.Unmarshal(message.Command().Data, &status)
	if err != nil {
		logger.Warn().Msg("Unexpected message")
		return nil
	}

	rooms := make([]roomStatus, 0, len(h.Rooms))

	for _, r := range h.Rooms {
		room := roomStatus{
			room: r,
		}
		tmp, err := getValue(status, r.CurrentTemperatureID, "state")
		if err != nil {
			continue
		}
		room.currentTemperature = tmp.(float64)
		tmp, err = getValue(status, r.TargetTemperatureID, "state")
		if err != nil {
			continue
		}
		room.targetTemperature = tmp.(float64)
		tmp, err = getValue(status, r.ThermalValveID, "state")
		if err != nil {
			continue
		}
		room.isHeating = tmp.(bool)
		tmp, err = getValue(status, r.ThermalValveID, "last_state_change")
		if err != nil {
			continue
		}
		var tmpTime types.TimeIndication
		types.Decode(tmp, &tmpTime)
		now := time.Now()
		room.lastHeatingStatusChange = time.Date(now.Year(), now.Month(), now.Day(), tmpTime.Hour, tmpTime.Minute, 0, 0, now.Location())
		if room.lastHeatingStatusChange.After(now) {
			room.lastHeatingStatusChange = room.lastHeatingStatusChange.Add(-24 * time.Hour)
		}
		rooms = append(rooms, room)
	}

	logger.Debug().Msg("Cycling through rooms")
	for _, r := range rooms {
		logger.Debug().Msgf("%v current:%v target:%v, is_heating:%v", r.room.ID, r.currentTemperature, r.targetTemperature, r.isHeating)
		if r.currentTemperature < r.targetTemperature-h.Threshold && !r.isHeating {
			command := types.DigitalCommand{
				Action:      types.SET,
				ApplianceID: r.room.ThermalValveID,
			}
			controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
			continue
		}
		timeDiff := time.Now().Sub(r.lastHeatingStatusChange)
		if r.currentTemperature > r.targetTemperature+h.Threshold && r.isHeating && (timeDiff < 15*time.Minute || timeDiff > MinimumRunningTime) {
			command := types.DigitalCommand{
				Action:      types.RESET,
				ApplianceID: r.room.ThermalValveID,
			}
			controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
		}
	}

	return nil
}

func (h *heating) String() string {
	return "heating"
}
