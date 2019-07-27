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
	"github.com/thoas/go-funk"
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
		var status []bus.DeviceDescription
		types.Decode(rawStatus, &status)
		if status == nil {
			status = make([]bus.DeviceDescription, 0)
		}
		deviceStatus := bus.DeviceDescription{}
		for _, a := range h.actuators {
			if device, err := a.Device(deviceID); err == nil {
				deviceStatus = device.Description()
			}
		}

		return funk.Map(status, func(d bus.DeviceDescription) bus.DeviceDescription {
			if d.ID == deviceID {
				return deviceStatus
			}
			return d
		})
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
	initialStatus := make([]bus.DeviceDescription, 0)
	for k, v := range h.actuators {
		v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
		for _, id := range v.DeviceIds() {
			// ignoring errors because we trust device apis
			device, _ := v.Device(id)
			initialStatus = append(initialStatus, device.Description())
		}
		h.listenForDeviceChanges(v.DeviceChanges())
	}
	logrus.Debug("Initial status: ", initialStatus)
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
