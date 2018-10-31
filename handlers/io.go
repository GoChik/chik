package handlers

import (
	"chik"
	"chik/actuator"
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
)

type io struct {
	actuator actuator.Actuator
	pins     map[int]bool
}

func NewIoHandler() chik.Handler {
	return &io{
		actuator.NewActuator(),
		map[int]bool{},
	}
}

func (h *io) Run(remote *chik.Controller) {
	logrus.Debug("starting io handler")
	h.actuator.Initialize()
	defer h.actuator.Deinitialize()

	in := remote.PubSub.Sub(chik.DigitalCommandType.String())
	for data := range in {
		message := data.(*chik.Message)

		command := chik.DigitalCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Error("cannot decode digital command: ", err)
			continue
		}

		if len(command.Command) != 1 {
			logrus.Error("Unexpected composed command")
			continue
		}

		h.pins[command.Pin] = true

		switch command.Command[0] {
		case chik.RESET:
			logrus.Debug("Turning off pin ", command.Pin)
			h.actuator.TurnOff(command.Pin)

		case chik.SET:
			logrus.Debug("Turning on pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)

		case chik.PUSH:
			logrus.Debug("Turning on and off pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)
			time.Sleep(1 * time.Second)
			h.actuator.TurnOff(command.Pin)

		case chik.TOGGLE:
			logrus.Debug("Switching pin ", command.Pin)
			if h.actuator.GetStatus(command.Pin) {
				h.actuator.TurnOff(command.Pin)
			} else {
				h.actuator.TurnOn(command.Pin)
			}
		}
	}

	logrus.Debug("shutting down io handler")
}

func (h *io) Status() interface{} {
	status := map[int]bool{}
	for pin := range h.pins {
		status[pin] = h.actuator.GetStatus(pin)
	}
	return status
}

func (h *io) String() string {
	return "io"
}
