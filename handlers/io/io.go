package io

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/platform"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "io").Logger()

type CurrentStatus struct {
	bus.DeviceDescription
	LastStateChange types.TimeIndication `json:"last_state_change" mapstructure:"last_state_change,string"`
}

type Status map[string]CurrentStatus
type io struct {
	chik.BaseHandler
	actuators     map[string]bus.Bus
	busByDevice   map[string]bus.Bus
	status        *chik.StatusHolder
	wg            sync.WaitGroup
	deviceChanges chan interface{}
}

// New creates a new IO handler
func New() chik.Handler {
	return &io{
		actuators:     platform.CreateBuses(),
		busByDevice:   make(map[string]bus.Bus),
		status:        chik.NewStatusHolder("io"),
		wg:            sync.WaitGroup{},
		deviceChanges: make(chan interface{}, 5),
	}
}

func (h *io) getDevice(deviceID string) (device bus.Device, err error) {
	if bus, ok := h.busByDevice[deviceID]; ok {
		device, err = bus.Device(deviceID)
		return
	}
	for _, actuator := range h.actuators {
		device, err = actuator.Device(deviceID)
		if err == nil {
			h.busByDevice[deviceID] = actuator
			return
		}
	}
	return
}

func (h *io) listenForDeviceChanges(channel <-chan string, controller *chik.Controller) {
	h.wg.Add(1)
	go func() {
		for device := range channel {
			h.deviceChanges <- device
		}
		h.wg.Done()
	}()
}

func (h *io) setStatus(controller *chik.Controller, applianceID string) {
	h.status.Edit(controller, func(rawStatus interface{}) (interface{}, bool) {
		var status Status
		types.Decode(rawStatus, &status)
		if status == nil {
			status = make(Status)
		}
		if device, err := h.getDevice(applianceID); err == nil {
			status[applianceID] = CurrentStatus{
				device.Description(),
				types.TimeIndication(time.Now().Unix()),
			}
		}
		return status, false
	})
}

func executeDigitalCommand(action types.Action, device bus.DigitalDevice, remote *chik.Controller) {
	switch action {
	case types.RESET:
		logger.Info().Msgf("Turning off %v", device.ID())
		device.TurnOff()

	case types.SET:
		logger.Info().Msgf("Turning on %v", device.ID())
		device.TurnOn()

	case types.PUSH:
		logger.Info().Msgf("Pushing %v", device.ID())
		device.TurnOn()
		time.Sleep(200 * time.Millisecond)
		device.TurnOff()

	case types.TOGGLE:
		logger.Info().Msgf("Toggling %v", device.ID())
		device.Toggle()

	default:
		logger.Warn().Msgf("Unknown action %v", action)
	}
}

func (h *io) parseDigitalCommand(remote *chik.Controller, message *chik.Message) {
	command := types.DigitalCommand{}
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Error().Msgf("Cannot decode digital command: %v", err)
		return
	}

	device, err := h.getDevice(command.ApplianceID)
	if err != nil {
		logger.Error().Msgf("Cannot find the specified device: %v", command.ApplianceID)
		return
	}

	switch device.Kind() {
	case bus.DigitalInputDevice, bus.DigitalOutputDevice:
		executeDigitalCommand(command.Action, device.(bus.DigitalDevice), remote)
		h.setStatus(remote, command.ApplianceID)

	default:
		logger.Warn().Msgf("Device %v does not support digital commands", command.ApplianceID)
	}
}

func (h *io) parseAnalogCommand(controller *chik.Controller, message *chik.Message) {
	var command types.AnalogCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Error().Msgf("Cannot decode analog command: %v", err)
		return
	}

	device, err := h.getDevice(command.ApplianceID)
	if err != nil {
		logger.Error().Msgf("Cannot find the specified device: %v", command.ApplianceID)
		return
	}
	if device.Kind() != bus.AnalogOutputDevice {
		logger.Error().Msgf("Device %v does not support analog set commands", command.ApplianceID)
		return
	}
	switch command.ValueType {
	case types.Absolute:
		device.(bus.AnalogDevice).SetValue(command.Value)

	case types.Relative:
		device.(bus.AnalogDevice).AddValue(command.Value)

	default:
		logger.Error().Msgf("Unsupported analog command value_type: %v", command.ValueType)
		return
	}

	h.setStatus(controller, command.ApplianceID)
}

func (h *io) Dependencies() []string {
	return []string{"status"}
}

func (h *io) Topics() []types.CommandType {
	return []types.CommandType{
		types.DigitalCommandType,
		types.AnalogCommandType,
	}
}

func (h *io) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	initialStatus := make(Status, 0)
	for k, v := range h.actuators {
		v.Initialize(config.Get(fmt.Sprintf("actuators.%s", k)))
		for _, id := range v.DeviceIds() {
			// ignoring errors because we trust device apis
			device, _ := v.Device(id)
			initialStatus[id] = CurrentStatus{
				device.Description(),
				types.TimeIndication(time.Now().Unix()),
			}
		}
		h.listenForDeviceChanges(v.DeviceChanges(), controller)
	}
	logger.Debug().Msgf("Initial status: %v", initialStatus)
	h.status.Set(initialStatus, controller)
	return chik.Interrupts{
		Timer: chik.NewEmptyTimer(),
		Event: h.deviceChanges,
	}, nil
}

func (h *io) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	switch message.Command().Type {
	case types.DigitalCommandType:
		h.parseDigitalCommand(controller, message)

	case types.AnalogCommandType:
		h.parseAnalogCommand(controller, message)
	}
	return nil
}

func (h *io) HandleChannelEvent(event interface{}, controller *chik.Controller) error {
	device, ok := event.(string)
	if !ok {
		return errors.New("Unexpected event received")
	}
	h.setStatus(controller, device)
	return nil
}

func (h *io) Teardown() {
	for _, v := range h.actuators {
		v.Deinitialize()
	}
	h.wg.Wait()
	close(h.deviceChanges)
}

func (h *io) String() string {
	return "io"
}
