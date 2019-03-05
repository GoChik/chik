// +build fake-actuator

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

type fakeActuator struct {
	devices map[string]*fakeDevice
}

func init() {
	actuators = append(actuators, newFakeActuator)
}

func (d *fakeDevice) ID() string {
	return d.Id
}

func (d *fakeDevice) TurnOn() {
	logrus.Debug("Turning on ", d.Id)
	d.Status = true
}

func (d *fakeDevice) TurnOff() {
	logrus.Debug("Turning off ", d.Id)
	d.Status = false
}

func (d *fakeDevice) Toggle() {
	logrus.Debug("Toggling on ", d.Id)
	d.Status = !d.Status
}

func (d *fakeDevice) GetStatus() bool {
	logrus.Debug("Get Status of ", d.Id)
	return d.Status
}

func (d *fakeDevice) StatusListener() chan bool {
	c := make(chan bool, 0)
	close(c)
	return c
}

func newFakeActuator() Actuator {
	return &fakeActuator{make(map[string]*fakeDevice)}
}

func (a *fakeActuator) Initialize(config interface{}) {
	logrus.Debug("Initialize called")
	var devices []*fakeDevice
	types.Decode(config, &devices)
	a.devices = funk.ToMap(devices, "Id").(map[string]*fakeDevice)
}

func (a *fakeActuator) Deinitialize() {
	logrus.Debug("Deinitialize called")
}

func (a *fakeActuator) Device(id string) (DigitalDevice, error) {
	device := a.devices[id]
	if device == nil {
		return nil, fmt.Errorf("No FAKE device with ID: %s found", id)
	}
	return device, nil
}

func (a *fakeActuator) DeviceIds() []string {
	result := make([]string, len(a.devices))
	for k := range a.devices {
		result = append(result, k)
	}
	return result
}

func (a *fakeActuator) String() string {
	return "fake"
}
