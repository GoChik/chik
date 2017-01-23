// +build gpio

package actuator

import (
	"iosomething/utils"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/davecheney/gpio"
)

type gpioActuator struct {
	mutex sync.Mutex
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	return &gpioActuator{sync.Mutex{}}
}

func (a *gpioActuator) Initialize()   {}
func (a *gpioActuator) Deinitialize() {}

func (a *gpioActuator) ExecuteCommand(command *utils.DigitalCommand) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	gpiopin, err := gpio.OpenPin(command.Pin, gpio.ModeOutput)
	if err != nil {
		logrus.Error("GPIO error:", err)
		return
	}
	defer gpiopin.Close()

	logrus.Debug("Executing command", command)

	switch command.Command {
	case utils.SWITCH_OFF:
		gpiopin.Clear()
		break

	case utils.SWITCH_ON:
		gpiopin.Set()
		break

	case utils.PUSH_BUTTON:
		gpiopin.Set()
		time.Sleep(1 * time.Second)
		gpiopin.Clear()
		break

	case utils.TOGGLE_ON_OFF:
		if gpiopin.Get() {
			gpiopin.Clear()
		} else {
			gpiopin.Set()
		}
		break
	}
}
