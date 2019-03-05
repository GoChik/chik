// +build gpio-actuator

package actuator

import (
	"fmt"
	"sync"

	"github.com/gochik/chik/types"
	"github.com/gochik/gpio"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

var mutex = sync.Mutex{}

type device struct {
	Id            string
	Number        uint
	Inverted      bool
	pin           gpio.Pin
	watcher       gpio.Watcher
	notifications []chan<- bool
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
	pin := gpio.NewOutput(d.Number, false)
	d.pin = pin

	if d.Inverted {
		d.pin.High()
	}
}

func (d *device) set(value bool) {
	mutex.Lock()
	defer mutex.Unlock()

	if value != d.Inverted {
		d.pin.High()
	} else {
		d.pin.Low()
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
	val, err := d.pin.Read()
	if err != nil {
		return false
	}
	return (val > 0) != d.Inverted
}

func (d *device) StatusListener() chan bool {
	c := make(chan bool, 0)
	go func() {
		for data := range d.watcher.Notification {
			c <- (data.Value > 0)
		}
	}()
	d.notifications = append(d.notifications, c)
	return c
}

func (a *gpioActuator) Initialize(conf interface{}) {
	var devices []*device
	err := types.Decode(conf, &devices)
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, device := range devices {
		logrus.Debug("New device for GPIO actuator: ", device)
		device.init()
		device.watcher.AddPin(device.Number)
	}
	a.devices = funk.ToMap(devices, "Id").(map[string]*device)
}

func (a *gpioActuator) Deinitialize() {
	for _, device := range a.devices {
		device.pin.Close()
		device.watcher.Close()
		for _, v := range device.notifications {
			close(v)
		}
	}
}

func (a *gpioActuator) Device(id string) (DigitalDevice, error) {
	device := a.devices[id]
	if device == nil {
		return nil, fmt.Errorf("No GPIO device with ID: %s found", id)
	}
	return device, nil
}

func (a *gpioActuator) DeviceIds() []string {
	return funk.Map(a.devices, func(k string, v *device) string {
		return k
	}).([]string)
}

func (a *gpioActuator) String() string {
	return "gpio"
}
