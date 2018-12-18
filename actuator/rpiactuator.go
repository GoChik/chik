// +build raspberrypi

package actuator

import (
	"strconv"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	rpio "github.com/stianeikeland/go-rpio"
)

type pin struct {
	rpio.Pin
	inverted bool
}

func (p pin) On() {
	if p.inverted {
		p.Low()
	} else {
		p.High()
	}
}

func (p pin) Off() {
	if p.inverted {
		p.High()
	} else {
		p.Low()
	}
}

type rpiActuator struct {
	mutex    sync.Mutex
	openPins map[int]*pin
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	return &rpiActuator{sync.Mutex{}, map[int]*pin{}}
}

func (a *rpiActuator) Initialize() {
	data := config.Get("gpio_actuator.pin_layout")
	if data == nil {
		config.Set("gpio_actuator.pin_layout", map[int]bool{0: false})
		config.Sync()
		logrus.Error("Cannot find gpio_actuator.pin_layout in config file, stub created")
		return
	}
	var pins map[string]bool
	err := types.Decode(data, &pins)
	if err != nil {
		logrus.Error(err)
		return
	}

	rpio.Open()
	for k, v := range pins {
		pinNumber, err := strconv.Atoi(k)
		if err != nil {
			logrus.Fatal("Failed to convert pin number to int: ", err)
		}
		a.openPins[pinNumber] = a.openPin(pinNumber, v)
	}
}

func (a *rpiActuator) Deinitialize() {
	rpio.Close()
}

func (a *rpiActuator) openPin(value int, inverted bool) *pin {
	if a.openPins[value] == nil {
		p := rpio.Pin(value)
		p.Output()
		a.openPins[value] = &pin{p, inverted}
	}
	return a.openPins[value]
}

func (a *rpiActuator) TurnOn(pin int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.openPins[pin].On()
}

func (a *rpiActuator) TurnOff(pin int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.openPins[pin].Off()
}

func (a *rpiActuator) GetStatus(pin int) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rpiPin := a.openPins[pin]
	if rpiPin.inverted {
		return rpiPin.Read() == rpio.Low
	}

	return rpiPin.Read() == rpio.High
}
