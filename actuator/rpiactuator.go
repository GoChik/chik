// +build raspberrypi

package actuator

import (
	"sync"

	rpio "github.com/stianeikeland/go-rpio"
)

type rpiActuator struct {
	mutex    sync.Mutex
	openPins map[int]rpio.Pin
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	return &rpiActuator{sync.Mutex{}, map[int]rpio.Pin{}}
}

func (a *rpiActuator) Initialize() {
	rpio.Open()
}

func (a *rpiActuator) Deinitialize() {
	rpio.Close()
}

func (a *rpiActuator) openPin(pin int) rpio.Pin {
	if a.openPins[pin] == 0 {
		p := rpio.Pin(pin)
		p.Output()
		a.openPins[pin] = p
	}
	return a.openPins[pin]
}

func (a *rpiActuator) TurnOn(pin int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rpiPin := a.openPin(pin)
	rpiPin.Low()
}

func (a *rpiActuator) TurnOff(pin int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rpiPin := a.openPin(pin)
	rpiPin.High()
}

func (a *rpiActuator) GetStatus(pin int) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	rpiPin := a.openPin(pin)
	return rpiPin.Read() == rpio.Low
}
