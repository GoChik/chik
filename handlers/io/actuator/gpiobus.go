/// +build gpioActuator

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
	Id       string
	Number   uint
	Inverted bool
	pin      gpio.Pin
}

type gpioBus struct {
	devices      map[string]*device
	devicesByPin map[uint]*device
	watcher      *gpio.Watcher
}

func init() {
	actuators = append(actuators, newGpioActuator)
}

func newGpioActuator() Bus {
	return &gpioBus{
		devices:      make(map[string]*device),
		devicesByPin: make(map[uint]*device),
		watcher:      gpio.NewWatcher(),
	}
}

func (d *device) init() {
	logrus.Debug("Opening pin ", d.Number, " with inverted logic: ", d.Inverted)
	pin, err := gpio.NewOutput(d.Number, false)
	if err != nil {
		logrus.Fatalf("Cannot open pin %d: %v", d.Number, err)
	}
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

func (d *device) Kind() DeviceKind {
	return DigitalOutputDevice
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

func (a *gpioBus) Initialize(conf interface{}) {
	var devices []*device
	err := types.Decode(conf, &devices)
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, device := range devices {
		logrus.Debug("New device for GPIO actuator: ", device)
		device.init()
		a.watcher.AddPin(device.Number)
	}
	a.devices = funk.ToMap(devices, "Id").(map[string]*device)
	a.devicesByPin = funk.ToMap(devices, "Number").(map[uint]*device)
}

func (a *gpioBus) Deinitialize() {
	for _, device := range a.devices {
		device.pin.Close()
	}
	a.watcher.Close()
}

func (a *gpioBus) Device(id string) (Device, error) {
	device := a.devices[id]
	if device == nil {
		return nil, fmt.Errorf("No GPIO device with ID: %s found", id)
	}
	return device, nil
}

func (a *gpioBus) DeviceIds() []string {
	return funk.Map(a.devices, func(k string, v *device) string {
		return k
	}).([]string)
}

func (a *gpioBus) DeviceChanges() <-chan string {
	c := make(chan string, 0)
	go func() {
		for data := range a.watcher.Notification {
			device := a.devicesByPin[data.Pin]
			c <- device.ID()
		}
		close(c)
	}()
	return c
}

func (a *gpioBus) String() string {
	return "gpio"
}
