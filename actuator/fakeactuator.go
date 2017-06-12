// +build fake

package actuator

import "github.com/Sirupsen/logrus"

type fakeActuator struct {
	initCalled bool
	pins       map[int]bool
}

func init() {
	NewActuator = NewFakeActuator
}

func NewFakeActuator() Actuator {
	return &fakeActuator{
		false,
		make(map[int]bool),
	}
}

func (a *fakeActuator) Initialize() {
	logrus.Debug("Initialize called")
	a.initCalled = true
}

func (a *fakeActuator) Deinitialize() {
	logrus.Debug("Deinitialize called")
	a.initCalled = false
}

func (a *fakeActuator) TurnOn(pin int) {
	logrus.Debug("FakeActuator: turn on pin: ", pin)
	a.pins[pin] = true
}

func (a *fakeActuator) TurnOff(pin int) {
	logrus.Debug("FakeActuator: turn off pin: ", pin)
	a.pins[pin] = false
}

func (a *fakeActuator) GetStatus(pin int) bool {
	return a.pins[pin]
}
