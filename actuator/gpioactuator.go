// +build gpio
//go:generate stringer -type=GPIOMode

package actuator

import (
	"fmt"
	"iosomething/utils"
	"log"
	"os"
	"sync"
	"time"
)

var mutex = sync.Mutex{}
var lastValues = make(map[int]bool) // pin, true: high, false: low
var pins = make(map[int]*GPIOPin)

type GPIOMode uint8

const (
	ModeInput GPIOMode = iota
	ModeOutput
)

type GPIOPin struct {
	pin  int
	mode GPIOMode
}

func init() {
	Initialize = func() {}
	Deinitialize = func() {}
	ExecuteCommand = executeCommand
}

func writefile(file string, text string) error {
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_SYNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = fd.Write([]byte(text))
	if err != nil {
		return err
	}

	return nil
}

func NewPin(number int, mode GPIOMode) (*GPIOPin, error) {
	// export gpio
	err := writefile("/sys/class/gpio/export", fmt.Sprintf("%d", number))
	if err != nil {
		return nil, err
	}

	// Write direction to file
	modestring := "in"
	if mode == ModeOutput {
		modestring = "out"
	}

	log.Println(modestring)

	err = writefile(fmt.Sprintf("/sys/devices/virtual/gpio/gpio%d/direction", number), modestring)
	if err != nil {
		return nil, err
	}

	return &GPIOPin{number, mode}, nil
}

func (g *GPIOPin) On() error {
	return writefile(fmt.Sprintf("/sys/devices/virtual/gpio/gpio%d/value", g.pin), "1")
}

func (g *GPIOPin) Off() error {
	return writefile(fmt.Sprintf("/sys/devices/virtual/gpio/gpio%d/value", g.pin), "0")
}

func executeCommand(command *utils.DigitalCommand) {
	mutex.Lock()
	defer mutex.Unlock()

	gpiopin := pins[command.Pin]
	if gpiopin == nil {
		gpiopin, err := NewPin(command.Pin, ModeOutput)
		if err != nil {
			fmt.Println(err)
			return
		}
		pins[command.Pin] = gpiopin
	}

	fmt.Println("Executing command", command)

	switch command.Command {
	case utils.SWITCH_OFF:
		gpiopin.Off()
		lastValues[gpiopin.pin] = false
		break

	case utils.SWITCH_ON:
		gpiopin.On()
		lastValues[gpiopin.pin] = true
		break

	case utils.PUSH_BUTTON:
		gpiopin.On()
		time.Sleep(1 * time.Second)
		gpiopin.Off()
		lastValues[gpiopin.pin] = false
		break

	case utils.TOGGLE_ON_OFF:
		oldValue := lastValues[gpiopin.pin]
		if oldValue == true {
			gpiopin.Off()
		} else {
			gpiopin.On()
		}
		lastValues[gpiopin.pin] = !oldValue
		break
	}
}
