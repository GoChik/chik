// +build raspberrypi-actuator

package actuator

import (
	"sync"

	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	rpio "github.com/stianeikeland/go-rpio"
	funk "github.com/thoas/go-funk"
)

var mutex = sync.Mutex{}

type device struct {
	Id       string
	Pin      rpio.Pin
	Inverted bool
}

type rpiActuator struct {
	devices map[string]*device
}

func init() {
	actuators = append(actuators, newRpiActuator)
}

func (d *device) ID() string {
	return d.Id
}

func (d *device) TurnOn() {
	mutex.Lock()
	if d.Inverted {
		d.Pin.Low()
	} else {
		d.Pin.High()
	}
	mutex.Unlock()
}

func (d *device) TurnOff() {
	mutex.Lock()
	if d.Inverted {
		d.Pin.High()
	} else {
		d.Pin.Low()
	}
	mutex.Unlock()
}

func (d *device) Toggle() {
	mutex.Lock()
	d.Pin.Toggle()
	mutex.Unlock()
}

func (d *device) GetStatus() bool {
	mutex.Lock()
	value := d.Pin.Read()
	mutex.Unlock()

	if d.Inverted {
		return value == rpio.Low
	}
	return value == rpio.High
}

func newRpiActuator() Actuator {
	return &rpiActuator{make(map[string]*device)}
}

func (a *rpiActuator) Initialize(conf interface{}) {
	var devices []*device
	err := types.Decode(conf, &devices)
	if err != nil {
		logrus.Error(err)
		return
	}

	rpio.Open()
	for _, v := range devices {
		v.Pin.Output()
		v.TurnOff()
	}

	a.devices = funk.ToMap(devices, "Id").(map[string]*device)
}

func (a *rpiActuator) Deinitialize() {
	rpio.Close()
}

func (a *rpiActuator) Device(id string) DigitalDevice {
	return a.devices[id]
}

func (a *rpiActuator) DeviceIds() []string {
	return funk.Map(a.devices, func(k string, v *device) string {
		return k
	}).([]string)
}

func (a *rpiActuator) String() string {
	return "raspberrypi"
}
