// +build gpio

package actuator

import (
	"chik/config"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/davecheney/gpio"
)

type pin struct {
	pin      gpio.Pin
	inverted bool
}

type gpioActuator struct {
	mutex    sync.Mutex
	openPins map[int]*pin
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	return &gpioActuator{
		sync.Mutex{},
		map[int]*pin{},
	}
}

func createPin(number int, inverted bool) *pin {
	logrus.Debug("Opening pin ", number, " with inverted logic: ", inverted)
	gpiopin, err := gpio.OpenPin(number, gpio.ModeOutput)
	if err != nil {
		logrus.Error("GPIO error:", err)
		return nil
	}

	if inverted {
		gpiopin.Set()
	}

	return &pin{
		gpiopin,
		inverted,
	}
}

func (a *gpioActuator) setPin(pin *pin, value bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if value != pin.inverted {
		pin.pin.Set()
	} else {
		pin.pin.Clear()
	}
}

func (a *gpioActuator) getPin(number int) *pin {
	gpiopin := a.openPins[number]
	if gpiopin == nil {
		gpiopin = createPin(number, false)
		a.openPins[number] = gpiopin
	}

	return gpiopin
}

func (a *gpioActuator) TurnOn(pin int) {
	a.setPin(a.getPin(pin), true)
}

func (a *gpioActuator) TurnOff(pin int) {
	a.setPin(a.getPin(pin), false)
}

func (a *gpioActuator) GetStatus(pin int) bool {
	gpiopin := a.getPin(pin)
	return gpiopin.pin.Get() != gpiopin.inverted
}

func (a *gpioActuator) Initialize() {
	pins, ok := config.Get("gpio_actuator.pin_layout").(map[int]bool)
	if !ok {
		config.Set("gpoi_actuator.pin_layout", map[int]bool{0: false})
		config.Sync()
		logrus.Error("Cannot find gpio_actuator.pin_layout in config file, stub created")
		return
	}

	for k, v := range pins {
		a.openPins[k] = createPin(k, v)
	}
}

func (a *gpioActuator) Deinitialize() {
	for _, v := range a.openPins {
		v.pin.Close()
	}
	a.openPins = map[int]*pin{}
}
