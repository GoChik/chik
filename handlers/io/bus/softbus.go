package bus

import (
	"fmt"

	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

type softDevice struct {
	Id       string
	SoftKind DeviceKind
	status   bool
}

type softBus struct {
	devices map[string]*softDevice
	updates chan string
}

func init() {
	actuators = append(actuators, newFakeBus)
}

func (d *softDevice) ID() string {
	return d.Id
}

func (d *softDevice) Kind() DeviceKind {
	return d.SoftKind
}

func (d *softDevice) Description() DeviceDescription {
	return DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: d.GetStatus(),
	}
}

func (d *softDevice) TurnOn() {
	logrus.Debug("Turning on ", d.Id)
	d.status = true
}

func (d *softDevice) TurnOff() {
	logrus.Debug("Turning off ", d.Id)
	d.status = false
}

func (d *softDevice) Toggle() {
	logrus.Debug("Toggling on ", d.Id)
	d.status = !d.status
}

func (d *softDevice) GetStatus() bool {
	logrus.Debug("Get Status of ", d.Id)
	return d.status
}

func newFakeBus() Bus {
	return &softBus{make(map[string]*softDevice), make(chan string, 0)}
}

func (a *softBus) Initialize(config interface{}) {
	logrus.Debug("Initialize called")
	var devices []*softDevice
	types.Decode(config, &devices)
	logrus.Debug("Devices found: ", len(devices))
	a.devices = funk.ToMap(devices, "Id").(map[string]*softDevice)
}

func (a *softBus) Deinitialize() {
	logrus.Debug("Deinitialize called")
	close(a.updates)
}

func (a *softBus) Device(id string) (Device, error) {
	logrus.Debug(id)
	device, ok := a.devices[id]
	if !ok {
		return nil, fmt.Errorf("No soft device with ID: %s found", id)
	}
	return device, nil
}

func (a *softBus) DeviceIds() []string {
	result := make([]string, 0, len(a.devices))
	for k := range a.devices {
		logrus.Debug(k)
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
