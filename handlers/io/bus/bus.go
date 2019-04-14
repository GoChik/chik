package actuator

import (
	"fmt"
)

var actuators []func() Bus

type DeviceKind uint8

const (
	None DeviceKind = iota
	DigitalInputDevice
	DigitalOutputDevice
	AnalogInputDevice
)

// Device is the interface every kind of device should implement
type Device interface {
	// Unique id for the device
	ID() string

	// Device type
	Kind() DeviceKind
}

// DigitalDevice is the interface that a binary input/output device should implement
type DigitalDevice interface {
	Device
	TurnOn()
	TurnOff()
	Toggle()
	GetStatus() bool
}

// AnalogDevice is the interface that an analog input device must implement
type AnalogDevice interface {
	Device
	GetValue() float32
}

// GetStatus returns the status of a device given his type
func GetStatus(device Device) interface{} {
	var status interface{}
	switch device.Kind() {
	case AnalogInputDevice:
		status = device.(AnalogDevice).GetValue()

	case DigitalInputDevice:
	case DigitalOutputDevice:
		status = device.(DigitalDevice).GetStatus()
	}
	return status
}

// CreateBuses creates the set of available actuators
func CreateBuses() map[string]Bus {
	result := make(map[string]Bus)
	for _, fun := range actuators {
		a := fun()
		result[a.String()] = a
	}
	return result
}

// Bus interface
type Bus interface {
	fmt.Stringer

	// Initialize initializes the actuator
	Initialize(config interface{})

	// Deinitialize is used when actuator is going off
	Deinitialize()

	// Given an unique id returns the corresponding device, an error if the id does not correspond to a device
	Device(id string) (Device, error)

	// list of device ids this bus is handling
	DeviceIds() []string

	// Channel that returns the id of a device in the moment the device changes his status
	DeviceChanges() <-chan string
}
