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
	Id     string
	Type   bus.DeviceKind
	status bool
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
	return bus.DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: d.GetStatus(),
	}
}

func (d *softDevice) TurnOn() {
	d.status = true
}

func (d *softDevice) TurnOff() {
	d.status = false
}

func (d *softDevice) Toggle() {
	d.status = !d.status
}

func (d *softDevice) GetStatus() bool {
	logger.Debug().Msgf("Get Status of ", d.Id)
	return d.status
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
