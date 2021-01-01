package heating

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

// Heating implements various logics needed to improve the heating system.
// Particularly the SizeFactor coefficent (from 1 to 100) improve
// condensing boiler running cycles by grouping together small zones
// to allow it to start once for multiple zones
// Eg:
// A room with SizeFactor of 100 starts alone.
// A room with SizeFactor of 50 starts with other zones till the total factor is >= 100

var logger = log.With().Str("handler", "heating").Logger()

type room struct {
	ID                   string `mapstructure:"id"`
	CurrentTemperatureID string `mapstructure:"current_temperature_id"`
	TargetTemperatureID  string `mapstructure:"target_temperature_id"`
	ThermalValveID       string `mapstructure:"thermal_valve_id"`
	SizeFactor           uint8  `mapstructure:"size_factor"`

	currentTemperature float64
	targetTemperature  float64
}

type heating struct {
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
		logger.Err(err)
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

func (h *heating) Setup(controller *chik.Controller) chik.Timer {
	for _, r := range h.Rooms {
		command := types.DigitalCommand{
			Action:      types.RESET,
			ApplianceID: r.ThermalValveID,
		}
		controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
	}
	return chik.NewEmptyTimer()
}

func getValue(status types.Status, id string) (interface{}, error) {
	v, ok := status["io"].(map[string]interface{})
	if !ok {
		return nil, errors.New("Cannot find io in status")
	}
	v, ok = v[id].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Cannot find thermal element %v in status", id)
	}
	return v["state"], nil
}

func (h *heating) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	var status types.Status
	err := json.Unmarshal(message.Command().Data, &status)
	if err != nil {
		logger.Warn().Msg("Unexpected message")
		return nil
	}

	for _, r := range h.Rooms {
		tmp, _ := getValue(status, r.CurrentTemperatureID)
		r.currentTemperature = tmp.(float64)
		tmp, _ = getValue(status, r.TargetTemperatureID)
		r.targetTemperature = tmp.(float64)
	}

	sort.Slice(h.Rooms, func(i, j int) bool {
		return (h.Rooms[i].currentTemperature - h.Rooms[i].targetTemperature) <
			(h.Rooms[j].currentTemperature - h.Rooms[j].targetTemperature)
	})

	currentSizeFactor := uint8(0)
	logger.Debug().Msg("Cycling through rooms")
	for _, r := range h.Rooms {
		logger.Debug().Msgf("%v current:%v target:%v, factor:%v", r.ID, r.currentTemperature, r.targetTemperature, currentSizeFactor)

		wantsToBeTurnedOn := r.currentTemperature-r.targetTemperature < 0 ||
			(currentSizeFactor > 0 && currentSizeFactor < 100)
		wantsToBeTurnedOff := (r.currentTemperature-r.targetTemperature > h.Threshold) && (currentSizeFactor == 0 || currentSizeFactor >= 100)
		isTurnedOn, err := getValue(status, r.ThermalValveID)
		if err != nil {
			logger.Error().Msgf("Cannot read Valve state: %v", err)
			return nil
		}

		if wantsToBeTurnedOn {
			currentSizeFactor += r.SizeFactor
			if !isTurnedOn.(bool) {
				command := types.DigitalCommand{
					Action:      types.SET,
					ApplianceID: r.ThermalValveID,
				}
				controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
			}
		}

		if wantsToBeTurnedOff && isTurnedOn.(bool) {
			command := types.DigitalCommand{
				Action:      types.RESET,
				ApplianceID: r.ThermalValveID,
			}
			controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
		}
	}

	return nil
}

func (h *heating) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *heating) Teardown() {}

func (h *heating) String() string {
	return "heating"
}
