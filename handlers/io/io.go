package io

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/io/actuator"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

type io struct {
	actuators     map[string]actuator.Actuator
	status        *chik.StatusHolder
	wg            sync.WaitGroup
	deviceChanges chan string
}

func New() chik.Handler {
	return &io{
		actuator.CreateActuators(),
		chik.NewStatusHolder("io"),
		sync.WaitGroup{},
		make(chan string, 0),
	}
}

func (h *io) listenForDeviceChanges(controller *chik.Controller, channel <-chan string) {
	h.wg.Add(1)
	go func() {
		for device := range channel {
			h.deviceChanges <- device
		}
		h.wg.Done()
	}()
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

func (h *io) parseMessage(remote *chik.Controller, message *chik.Message) {
	command := types.DigitalCommand{}
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logrus.Error("cannot decode digital command: ", err)
		return
	}

	if len(command.Command) != 1 {
		logrus.Error("Unexpected command length: ", len(command.Command))
		return
	}

	for _, actuator := range h.actuators {
		device, err := actuator.Device(command.DeviceID)
		if err == nil {
			execute(command.Command[0], device, remote)
			h.setStatus(remote, command.DeviceID)
		}
	}
}

func (h *io) tearDown() {
	for _, v := range h.actuators {
		v.Deinitialize()
	}
	h.wg.Wait()
	close(h.deviceChanges)
}

func (h *io) Run(remote *chik.Controller) {
	logrus.Debug("starting io handler")
	{
		initialStatus := make(map[string]bool)
		for k, v := range h.actuators {
			v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
			h.listenForDeviceChanges(remote, v.DeviceChanges())
			for _, id := range v.DeviceIds() {
				// ignoring errors because we trust device apis
				device, _ := v.Device(id)
				initialStatus[id] = device.GetStatus()
			}
		}
		h.status.Set(initialStatus, remote)
	}

	defer h.tearDown()

	in := remote.Sub(types.DigitalCommandType.String())
	for {
		select {
		case data, ok := <-in:
			if !ok {
				return
			}
			h.parseMessage(remote, data.(*chik.Message))

		case deviceID := <-h.deviceChanges:
			h.setStatus(remote, deviceID)
		}
	}
}

func (h *io) String() string {
	return "io"
}