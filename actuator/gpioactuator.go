// +build gpio
//go:generate stringer -type=GPIOMode

package actuator

import (
	"iosomething/utils"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/davecheney/gpio"
)

var mutex = sync.Mutex{}

func init() {
	Initialize = func() {}
	Deinitialize = func() {}
	ExecuteCommand = executeCommand
}

func executeCommand(command *utils.DigitalCommand) {
	mutex.Lock()
	defer mutex.Unlock()

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
		if true {
			gpiopin.Clear()
		} else {
			gpiopin.Set()
		}
		break
	}

}
