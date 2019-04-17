package io

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
)

type io struct {
	actuators     map[string]bus.Bus
	status        *chik.StatusHolder
	wg            sync.WaitGroup
	deviceChanges chan string
}

func New() chik.Handler {
	return &io{
		bus.CreateBuses(),
		chik.NewStatusHolder("io"),
		sync.WaitGroup{},
		make(chan string, 0),
	}
}

func (h *io) listenForDeviceChanges(channel <-chan string) {
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
		var status map[string]interface{}
		types.Decode(rawStatus, &status)
		if status == nil {
			status = make(map[string]interface{})
		}
		for _, a := range h.actuators {
			if device, err := a.Device(deviceID); err == nil {
				status[deviceID] = bus.GetStatus(device)
			}
		}
		return status
	})
}

func executeDigitalCommand(command types.Action, device bus.DigitalDevice, remote *chik.Controller) {
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

	for _, a := range h.actuators {
		device, err := a.Device(command.DeviceID)
		if err == nil {
			switch device.Kind() {
			case bus.DigitalInputDevice:
			case bus.DigitalOutputDevice:
				executeDigitalCommand(command.Command[0], device.(bus.DigitalDevice), remote)

			case bus.AnalogInputDevice:
				logrus.Warn("Analog commands are not yet implemented")
			}
			h.setStatus(remote, command.DeviceID)
		}
	}
}

func (h *io) setup(remote *chik.Controller) {
	initialStatus := make(map[string]interface{})
	for k, v := range h.actuators {
		v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
		h.listenForDeviceChanges(v.DeviceChanges())
		for _, id := range v.DeviceIds() {
			// ignoring errors because we trust device apis
			device, _ := v.Device(id)
			initialStatus[id] = bus.GetStatus(device)
		}
	}
	h.status.Set(initialStatus, remote)
}

func (h *io) tearDown() {
	for _, v := range h.actuators {
		v.Deinitialize()
	}
	h.wg.Wait()
	close(h.deviceChanges)
}

func (h *io) Run(remote *chik.Controller) {
	h.setup(remote)
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
