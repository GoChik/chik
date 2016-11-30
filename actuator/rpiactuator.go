// +build raspberrypi

package actuator

import (
	"log"
	"sync"
	"time"

	"iosomething/utils"

	rpio "github.com/stianeikeland/go-rpio"
)

var mutex = sync.Mutex{}

func init() {
	Initialize = initialize
	Deinitialize = deinitialize
	ExecuteCommand = executeCommand
}

func initialize() {
	rpio.Open()
}

func deinitialize() {
	rpio.Close()
}

func executeCommand(command *utils.DigitalCommand) {
	mutex.Lock()
	defer mutex.Unlock()

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
