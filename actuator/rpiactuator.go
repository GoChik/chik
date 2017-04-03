// +build raspberrypi

package actuator

import (
	"encoding/json"
	"iosomething"
	"log"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	rpio "github.com/stianeikeland/go-rpio"
)

type rpiActuator struct {
	mutex sync.Mutex
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	return &rpiActuator{sync.Mutex{}}
}

func (a *rpiActuator) Initialize() {
	rpio.Open()
}

func (a *rpiActuator) Deinitialize() {
	rpio.Close()
}

func (a *rpiActuator) Execute(data []byte) (reply []byte) {
	command := iosomething.DigitalCommand{}
	err := json.Unmarshal(data, &command)
	if err != nil {
		logrus.Error("Error parsing command", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	log.Println("Executing command", command)

	rpiPin := rpio.Pin(command.Pin)
	rpiPin.Output()

	switch command.Command {
	case iosomething.PUSH_BUTTON:
		rpiPin.Low()
		time.Sleep(1 * time.Second)
		rpiPin.High()
		break

	case iosomething.TOGGLE_ON_OFF:
		rpiPin.Toggle()
		break

	case iosomething.SWITCH_ON:
		rpiPin.High()
		break

	case iosomething.SWITCH_OFF:
		rpiPin.Low()
		break

	case iosomething.GET_STATUS:
		data, err := json.Marshal(iosomething.StatusIndication{
			command.Pin,
			rpiPin.Read() == rpio.High,
		})

		if err != nil {
			logrus.Error("Error encoding reply")
			return
		}
		reply = data
		break
	}
	return
}
