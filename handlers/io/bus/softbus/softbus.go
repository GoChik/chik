package softbus

import (
	"fmt"

	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	funk "github.com/thoas/go-funk"
)

var logger = log.With().Str("handler", "io").Str("bus", "soft").Logger()

type softDevice struct {
	Id    string
	Type  bus.DeviceKind
	Value interface{}
}

type softBus struct {
	devices map[string]*softDevice
	updates chan string
}

func New() bus.Bus {
	return &softBus{make(map[string]*softDevice), make(chan string, 0)}
}

func (d *softDevice) ID() string {
	return d.Id
}

func (d *softDevice) Kind() bus.DeviceKind {
	return d.Type
}

func (d *softDevice) Description() bus.DeviceDescription {
	var state interface{}
	switch d.Kind() {
	case bus.DigitalInputDevice, bus.DigitalOutputDevice:
		if d.Value == nil {
			state = false
		} else {
			state = d.Value.(bool)
		}

	case bus.AnalogInputDevice, bus.AnalogOutputDevice:
		if d.Value == nil {
			state = float64(0)
		} else {
			state = d.Value.(float64)
		}

	default:
		state = false
	}
	return bus.DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: state,
	}
}

func (d *softDevice) TurnOn() {
	if d.Type != bus.DigitalOutputDevice {
		logger.Error().Msgf("Cannot turn on %v, it is not a digital output device", d.Id)
		return
	}
	d.Value = true
}

func (d *softDevice) TurnOff() {
	if d.Type != bus.DigitalOutputDevice {
		logger.Error().Msgf("Cannot turn off %v, it is not a digital output device", d.Id)
		return
	}
	d.Value = false
}

func (d *softDevice) Toggle() {
	if d.Type != bus.DigitalOutputDevice {
		logger.Error().Msgf("Cannot toggle %v, it is not a digital output device", d.Id)
		return
	}
	d.Value = !d.Value.(bool)
}

func (d *softDevice) SetValue(value float64) {
	if d.Type != bus.AnalogOutputDevice {
		logger.Error().Msgf("Cannot set value on %v, it is not an analog output device", d.Id)
		return
	}
	d.Value = value
}

func (a *softBus) Initialize(config interface{}) {
	var devices []*softDevice
	err := types.Decode(config, &devices)
	if err != nil {
		logger.Error().Msgf("Failed initializing bus: %v", err)
	}
	a.devices = funk.ToMap(devices, "Id").(map[string]*softDevice)
}

func (a *softBus) Deinitialize() {
	logger.Debug().Msg("Deinitialize called")
	close(a.updates)
}

func (a *softBus) Device(id string) (bus.Device, error) {
	device, ok := a.devices[id]
	if !ok {
		return nil, fmt.Errorf("No soft device with ID: %s found", id)
	}
	return device, nil
}

func (a *softBus) DeviceIds() []string {
	result := make([]string, 0, len(a.devices))
	for k := range a.devices {
		result = append(result, k)
	}
	return result
}

func (a *softBus) DeviceChanges() <-chan string {
	return a.updates
}

func (a *softBus) String() string {
	return "soft"
}
