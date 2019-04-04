/// +build fakeActuator

package actuator

import (
	"fmt"

	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

type fakeDevice struct {
	Id     string
	status bool
}

type fakeBus struct {
	devices map[string]*fakeDevice
}

func init() {
	actuators = append(actuators, newFakeBus)
}

func (d *fakeDevice) ID() string {
	return d.Id
}

func (d *fakeDevice) Kind() DeviceKind {
	return DigitalOutputDevice
}

func (d *fakeDevice) TurnOn() {
	logrus.Debug("Turning on ", d.Id)
	d.status = true
}

func (d *fakeDevice) TurnOff() {
	logrus.Debug("Turning off ", d.Id)
	d.status = false
}

func (d *fakeDevice) Toggle() {
	logrus.Debug("Toggling on ", d.Id)
	d.status = !d.status
}

func (d *fakeDevice) GetStatus() bool {
	logrus.Debug("Get Status of ", d.Id)
	return d.status
}

func newFakeBus() Bus {
	return &fakeBus{make(map[string]*fakeDevice)}
}

func (a *fakeBus) Initialize(config interface{}) {
	logrus.Debug("Initialize called")
	var devices []*fakeDevice
	types.Decode(config, &devices)
	logrus.Debug("Devices found: ", len(devices))
	a.devices = funk.ToMap(devices, "Id").(map[string]*fakeDevice)
}

func (a *fakeBus) Deinitialize() {
	logrus.Debug("Deinitialize called")
}

func (a *fakeBus) Device(id string) (Device, error) {
	logrus.Debug(id)
	device := a.devices[id]
	if device == nil {
		logrus.Debug("device not found")
		return nil, fmt.Errorf("No FAKE device with ID: %s found", id)
	}
	return device, nil
}

func (a *fakeBus) DeviceIds() []string {
	result := make([]string, 0, len(a.devices))
	for k := range a.devices {
		logrus.Debug(k)
		result = append(result, k)
	}
	return result
}

func (a *fakeBus) DeviceChanges() <-chan string {
	c := make(chan string, 0)
	close(c)
	return c
}

func (a *fakeBus) String() string {
	return "fake"
}
