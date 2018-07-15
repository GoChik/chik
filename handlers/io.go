package handlers

import (
	"encoding/json"
	"iosomething"
	"iosomething/actuator"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/satori/go.uuid"
)

type io struct {
	id       uuid.UUID
	actuator actuator.Actuator
	pins     map[int]bool
}

func NewIoHandler(uuid uuid.UUID) iosomething.Handler {
	return &io{
		uuid,
		actuator.NewActuator(),
		map[int]bool{},
	}
}

func (h *io) HandlerRoutine(remote *iosomething.Remote) {
	logrus.Debug("starting io handler")
	h.actuator.Initialize()
	defer h.actuator.Deinitialize()

	in := remote.PubSub.Sub(iosomething.DigitalCommandType.String())
	for data := range in {
		message := data.(*iosomething.Message)

		command := iosomething.DigitalCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Error("cannot decode digital command: ", err)
			continue
		}

		h.pins[command.Pin] = true

		switch command.Command {
		case iosomething.SWITCH_OFF:
			logrus.Debug("Turning off pin ", command.Pin)
			h.actuator.TurnOff(command.Pin)

		case iosomething.SWITCH_ON:
			logrus.Debug("Turning on pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)

		case iosomething.PUSH_BUTTON:
			logrus.Debug("Turning on and off pin ", command.Pin)
			h.actuator.TurnOn(command.Pin)
			time.Sleep(1 * time.Second)
			h.actuator.TurnOff(command.Pin)

		case iosomething.TOGGLE_ON_OFF:
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
