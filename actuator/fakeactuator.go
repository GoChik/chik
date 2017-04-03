// +build fake

package actuator

import (
	"encoding/json"
	"iosomething"

	"github.com/Sirupsen/logrus"
)

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

func (a *fakeActuator) Execute(data []byte) (reply []byte) {
	command := iosomething.DigitalCommand{}
	err := json.Unmarshal(data, &command)
	if err != nil {
		logrus.Error("Error parsing command", err)
		return
	}

	switch command.Command {
	case iosomething.PUSH_BUTTON:
		logrus.Debug("PUSH_BUTTON, pin: ", command.Pin)
		a.pins[command.Pin] = false
		break

	case iosomething.SWITCH_ON:
		logrus.Debug("SWITCH_ON, pin: ", command.Pin)
		a.pins[command.Pin] = true
		break

	case iosomething.SWITCH_OFF:
		logrus.Debug("SWITCH_OFF, pin: ", command.Pin)
		a.pins[command.Pin] = false
		break

	case iosomething.TOGGLE_ON_OFF:
		logrus.Debug("TOGGLE_ON_OFF, pin: ", command.Pin)
		if a.pins[command.Pin] {
			a.pins[command.Pin] = false
		} else {
			a.pins[command.Pin] = true
		}
		break

	case iosomething.GET_STATUS:
		logrus.Debug("GET_STATUS, pin: ", command.Pin)
		reply, err = json.Marshal(iosomething.StatusIndication{command.Pin, a.pins[command.Pin]})
		if err != nil {
			logrus.Error("Error encoding json reply: ", err)
		}
		break
	}
	return
}
