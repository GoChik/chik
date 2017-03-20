// +build gpio

package actuator

import (
	"encoding/json"
	"iosomething/utils"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/davecheney/gpio"
)

var configFile = "gpio.json"

type gpioConf struct {
	Pins map[int]bool
}

type pin struct {
	pin      gpio.Pin
	inverted bool
}

type gpioActuator struct {
	mutex    sync.Mutex
	openPins map[int]*pin
}

func init() {
	NewActuator = newActuator
}

func newActuator() Actuator {
	actuator := gpioActuator{
		sync.Mutex{},
		map[int]*pin{},
	}

	confPath := utils.GetConfPath(configFile)
	if confPath != "" {
		conf := gpioConf{}
		err := utils.ParseConf(confPath, &conf)
		if err != nil {
			logrus.Error("Cannot parse actuator configuration: ", err)
			return &actuator
		}

		for k, v := range conf.Pins {
			actuator.openPins[k] = createPin(k, v)
		}
	}

	return &actuator
}

func createPin(number int, inverted bool) *pin {
	logrus.Debug("Opening pin ", number, " with inverted logic: ", inverted)
	gpiopin, err := gpio.OpenPin(number, gpio.ModeOutput)
	if err != nil {
		logrus.Error("GPIO error:", err)
		return nil
	}

	if inverted {
		gpiopin.Set()
	}

	return &pin{
		gpiopin,
		inverted,
	}
}

func (a *gpioActuator) setPin(pin *pin, value bool) {
	if value != pin.inverted {
		pin.pin.Set()
	} else {
		pin.pin.Clear()
	}
}

func (a *gpioActuator) Initialize() {}
func (a *gpioActuator) Deinitialize() {
	for _, v := range a.openPins {
		v.pin.Close()
	}
}

func (a *gpioActuator) Execute(data []byte) (reply []byte) {
	command := DigitalCommand{}
	err := json.Unmarshal(data, &command)
	if err != nil {
		logrus.Error("Error parsing command", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	gpiopin := a.openPins[command.Pin]
	if gpiopin == nil {
		gpiopin = createPin(command.Pin, false)
		a.openPins[command.Pin] = gpiopin
	}

	logrus.Debug("Executing command", command)

	switch command.Command {
	case SWITCH_OFF:
		a.setPin(gpiopin, false)
		break

	case SWITCH_ON:
		a.setPin(gpiopin, true)
		break

	case PUSH_BUTTON:
		a.setPin(gpiopin, true)
		time.Sleep(1 * time.Second)
		a.setPin(gpiopin, false)
		break

	case TOGGLE_ON_OFF:
		if gpiopin.pin.Get() || (!gpiopin.pin.Get() && gpiopin.inverted) {
			a.setPin(gpiopin, false)
		} else {
			a.setPin(gpiopin, true)
		}
		break

	case GET_STATUS:
		data, err = json.Marshal(StatusIndication{
			command.Pin,
			gpiopin.pin.Get() != gpiopin.inverted,
		})

		if err != nil {
			logrus.Error("Unable to encode reply message")
			return
		}
		reply = data
		break
	}
	return
}
