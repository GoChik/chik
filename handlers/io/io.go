package io

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/platform"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
)

type ioStatus map[string]bus.DeviceDescription
type io struct {
	actuators map[string]bus.Bus
	status    *chik.StatusHolder
	wg        sync.WaitGroup
}

type ioDeviceChanges struct {
	DeviceID string
}

// New creates a new IO handler
func New() chik.Handler {
	return &io{
		platform.CreateBuses(),
		chik.NewStatusHolder("io"),
		sync.WaitGroup{},
	}
}

func (h *io) listenForDeviceChanges(channel <-chan string, controller *chik.Controller) {
	h.wg.Add(1)
	go func() {
		for device := range channel {
			controller.Pub(types.NewCommand(types.IODeviceStatusChangedCommandType, ioDeviceChanges{device}),
				chik.LoopbackID)
		}
		h.wg.Done()
	}()
}

func (h *io) setStatus(controller *chik.Controller, applianceID string) {
	h.status.Edit(controller, func(rawStatus interface{}) interface{} {
		var status ioStatus
		types.Decode(rawStatus, &status)
		if status == nil {
			status = make(ioStatus)
		}
		for _, a := range h.actuators {
			if device, err := a.Device(applianceID); err == nil {
				status[applianceID] = device.Description()
			}
		}

		return status
	})
}

func executeDigitalCommand(action types.Action, device bus.DigitalDevice, remote *chik.Controller) {
	switch action {
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

	default:
		logrus.Warningf("Unknown action %v", action)
	}
}

func (h *io) parseDigitalCommand(remote *chik.Controller, message *chik.Message) {
	command := types.DigitalCommand{}
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logrus.Error("cannot decode digital command: ", err)
		return
	}

	for _, a := range h.actuators {
		device, err := a.Device(command.ApplianceID)
		if err == nil {
			switch device.Kind() {
			case bus.DigitalInputDevice:
			case bus.DigitalOutputDevice:
				executeDigitalCommand(command.Action, device.(bus.DigitalDevice), remote)

			case bus.AnalogInputDevice:
				logrus.Warn("Analog commands are not yet implemented")
			}
			h.setStatus(remote, command.ApplianceID)
		}
	}
}

func (h *io) Dependencies() []string {
	return []string{"status"}
}

func (h *io) Topics() []types.CommandType {
	return []types.CommandType{
		types.DigitalCommandType,
		types.IODeviceStatusChangedCommandType,
	}
}

func (h *io) Setup(controller *chik.Controller) chik.Timer {
	initialStatus := make(ioStatus, 0)
	for k, v := range h.actuators {
		v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
		for _, id := range v.DeviceIds() {
			// ignoring errors because we trust device apis
			device, _ := v.Device(id)
			initialStatus[id] = device.Description()
		}
		h.listenForDeviceChanges(v.DeviceChanges(), controller)
	}
	logrus.Debug("Initial status: ", initialStatus)
	h.status.Set(initialStatus, controller)
	return chik.NewEmptyTimer()
}

func (h *io) HandleMessage(message *chik.Message, controller *chik.Controller) {
	switch message.Command().Type {
	case types.DigitalCommandType:
		h.parseDigitalCommand(controller, message)

	case types.IODeviceStatusChangedCommandType:
		var data ioDeviceChanges
		err := json.Unmarshal(message.Command().Data, &data)
		if err != nil {
			logrus.Warn("Cannot parse device update ", err)
		}
		h.setStatus(controller, data.DeviceID)
	}
}

func (h *io) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *io) Teardown() {
	for _, v := range h.actuators {
		v.Deinitialize()
	}
	h.wg.Wait()
}

func (h *io) String() string {
	return "io"
}
