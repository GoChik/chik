// +build raspberrypi

package actuator

import (
	"iosomething/utils"
	"log"
	"sync"
	"time"

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

func (a *rpiActuator) ExecuteCommand(command *utils.DigitalCommand) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	log.Println("Executing command", command)

	rpiPin := rpio.Pin(command.Pin)
	rpiPin.Output()

	switch command.Command {
	case utils.PUSH_BUTTON:
		rpiPin.Low()
		time.Sleep(1 * time.Second)
		rpiPin.High()
		break

	case utils.TOGGLE_ON_OFF:
		rpiPin.Toggle()
		break

	case utils.SWITCH_ON:
		rpiPin.High()
		break

	case utils.SWITCH_OFF:
		rpiPin.Low()
		break
	}
}
