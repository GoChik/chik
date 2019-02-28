// +build gpio-actuator

package actuator

import (
	"sync"

	"github.com/davecheney/gpio"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

var mutex = sync.Mutex{}

type device struct {
	Id       string
	Number   int
	Inverted bool
	pin      gpio.Pin
}

type gpioActuator struct {
	devices map[string]*device
}

func init() {
	actuators = append(actuators, newGpioActuator)
}

func newGpioActuator() Actuator {
	return &gpioActuator{make(map[string]*device)}
}

func (d *device) init() {
	logrus.Debug("Opening pin ", d.Number, " with inverted logic: ", d.Inverted)
	pin, err := gpio.OpenPin(d.Number, gpio.ModeOutput)
	if err != nil {
		logrus.Error("GPIO error:", err)
		return
	}
	d.pin = pin

	if d.Inverted {
		d.pin.Set()
	}
}

func (d *device) set(value bool) {
	mutex.Lock()
	defer mutex.Unlock()

	if value != d.Inverted {
		d.pin.Set()
	} else {
		d.pin.Clear()
	}
}

func (d *device) ID() string {
	return d.Id
}

func (d *device) TurnOn() {
	d.set(true)
}

func (d *device) TurnOff() {
	d.set(false)
}

func (d *device) Toggle() {
	d.set(!d.GetStatus())
}

func (d *device) GetStatus() bool {
	return d.pin.Get() != d.Inverted
}

func (a *gpioActuator) Initialize(conf interface{}) {
	var devices []*device
	err := types.Decode(conf, &devices)
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, device := range devices {
		device.init()
	}
	a.devices = funk.ToMap(devices, "Id").(map[string]*device)
}

func (a *gpioActuator) Deinitialize() {
	for _, device := range a.devices {
		device.pin.Close()
	}
}

func (a *gpioActuator) Device(id string) DigitalDevice {
	return a.devices[id]
}

func (a *gpioActuator) DeviceIds() []string {
	return funk.Map(a.devices, func(k string, v *device) string {
		return k
	}).([]string)
}

func (a *gpioActuator) String() string {
	return "gpio"
}
