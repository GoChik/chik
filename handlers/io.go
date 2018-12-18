package handlers

import (
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gochik/chik"
	"github.com/gochik/chik/actuator"
	"github.com/gochik/chik/types"
)

type io struct {
	actuator actuator.Actuator
	pins     map[int]bool
	status   *chik.StatusHolder
}

func NewIoHandler() chik.Handler {
	return &io{
		actuator.NewActuator(),
		map[int]bool{},
		chik.NewStatusHolder("io"),
	}
}

func (h *io) Run(remote *chik.Controller) {
	logrus.Debug("starting io handler")
	h.actuator.Initialize()
	defer h.actuator.Deinitialize()

	h.status.SetStatus(h.Status(), remote)

	in := remote.Sub(types.DigitalCommandType.String())
	for data := range in {
		message := data.(*chik.Message)

		command := types.DigitalCommand{}
		err := json.Unmarshal(message.Command().Data, &command)
		if err != nil {
			logrus.Error("cannot decode digital command: ", err)
			continue
		}

		if len(command.Command) != 1 {
			logrus.Error("Unexpected composed command")
			continue
		}

		h.pins[command.Pin] = true

		switch types.Action(command.Command[0]) {
		case types.RESET:
			logrus.Debug("Turning off pin ", command.Pin)
			h.actuator.TurnOff(command.Pin)

		case types.SET:
			logrus.Debug("Turning on pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)

		case types.PUSH:
			logrus.Debug("Turning on and off pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)
			time.Sleep(1 * time.Second)
			h.actuator.TurnOff(command.Pin)

		case types.TOGGLE:
			logrus.Debug("Switching pin ", command.Pin)
			if h.actuator.GetStatus(command.Pin) {
				h.actuator.TurnOff(command.Pin)
			} else {
				h.actuator.TurnOn(command.Pin)
			}
		}

		h.status.SetStatus(h.Status(), remote)
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
