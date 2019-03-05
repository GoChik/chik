package io

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/io/actuator"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

type io struct {
	actuators map[string]actuator.Actuator
	status    *chik.StatusHolder
}

func New() chik.Handler {
	return &io{
		actuator.CreateActuators(),
		chik.NewStatusHolder("io"),
	}
}

func (h *io) setStatus(controller *chik.Controller, deviceID string) {
	h.status.Edit(controller, func(rawStatus interface{}) interface{} {
		var status map[string]bool
		types.Decode(rawStatus, &status)
		if status == nil {
			status = make(map[string]bool)
		}
		res := funk.Map(status, func(k string, v bool) (string, bool) {
			if k == deviceID {
				for _, actuator := range h.actuators {
					if device, err := actuator.Device(deviceID); err == nil {
						v = device.GetStatus()
					}
				}
			}
			return k, v
		})
		return res
	})
}

func execute(command types.Action, device actuator.DigitalDevice, remote *chik.Controller) {
	switch types.Action(command) {
	case types.RESET:
		logrus.Debug("Turning off device ", device.ID())
		device.TurnOff()

	case types.SET:
		logrus.Debug("Turning on device ", device.ID())
		device.TurnOn()

	case types.PUSH:
		logrus.Debug("Pushing device ", device.ID())
		device.TurnOn()
		time.Sleep(500 * time.Millisecond)
		device.TurnOff()

	case types.TOGGLE:
		logrus.Debug("Toggling device ", device.ID())
		device.Toggle()
	}
}

func (h *io) Run(remote *chik.Controller) {
	logrus.Debug("starting io handler")
	{
		initialStatus := make(map[string]bool)
		for k, v := range h.actuators {
			v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
			for _, id := range v.DeviceIds() {
				// ignoring errors because we trust device apis
				device, _ := v.Device(id)
				initialStatus[id] = device.GetStatus()
			}
		}
		h.status.Set(initialStatus, remote)
	}

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
			logrus.Error("Unexpected command length: ", len(command.Command))
			continue
		}

		for _, actuator := range h.actuators {
			device, err := actuator.Device(command.DeviceID)
			if err == nil {
				execute(command.Command[0], device, remote)
				h.setStatus(remote, command.DeviceID)
			}
		}
	}

	for _, v := range h.actuators {
		v.Deinitialize()
	}
}

func (h *io) String() string {
	return "io"
}
