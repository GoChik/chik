// +build raspberrypi

package actuator

import (
	"encoding/json"
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
	command := DigitalCommand{}
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
	case PUSH_BUTTON:
		rpiPin.Low()
		time.Sleep(1 * time.Second)
		rpiPin.High()
		break

	case TOGGLE_ON_OFF:
		rpiPin.Toggle()
		break

	case SWITCH_ON:
		rpiPin.High()
		break

	case SWITCH_OFF:
		rpiPin.Low()
		break

	case GET_STATUS:
		data, err := json.Marshal(StatusIndication{
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
