package softbus

import (
	"fmt"

	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

var log = logrus.WithFields(logrus.Fields{
	"handler": "io",
	"bus":     "soft",
})

type softDevice struct {
	Id       string
	SoftKind bus.DeviceKind
	status   bool
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
	return d.SoftKind
}

func (d *softDevice) Description() bus.DeviceDescription {
	return bus.DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: d.GetStatus(),
	}
}

func (d *softDevice) TurnOn() {
	log.Debug("Turning on ", d.Id)
	d.status = true
}

func (d *softDevice) TurnOff() {
	log.Debug("Turning off ", d.Id)
	d.status = false
}

func (d *softDevice) Toggle() {
	log.Debug("Toggling on ", d.Id)
	d.status = !d.status
}

func (d *softDevice) GetStatus() bool {
	log.Debug("Get Status of ", d.Id)
	return d.status
}

func (a *softBus) Initialize(config interface{}) {
	log.Debug("Initialize called")
	var devices []*softDevice
	types.Decode(config, &devices)
	log.Debug("Devices found: ", len(devices))
	a.devices = funk.ToMap(devices, "Id").(map[string]*softDevice)
}

func (a *softBus) Deinitialize() {
	log.Debug("Deinitialize called")
	close(a.updates)
}

func (a *softBus) Device(id string) (bus.Device, error) {
	log.Debug(id)
	device, ok := a.devices[id]
	if !ok {
		return nil, fmt.Errorf("No soft device with ID: %s found", id)
	}
	return device, nil
}

func (a *softBus) DeviceIds() []string {
	result := make([]string, 0, len(a.devices))
	for k := range a.devices {
		log.Debug(k)
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
