package bus

import (
	"fmt"
)

var Actuators []func() Bus

type DeviceKind uint8

// device types
// None: not assigned or unknown
// DigitalInputDevice: a digital device that can be only read (eg: wall switch)
// DigitalOutputDevice: a digital device that can be written (eg: light bulb)
// AnalogInputDevice: an analog device that can be read (eg: dimmer)
// AnalogOutputDevice: an analog device that can be written (eg: room temperature)
const (
	None DeviceKind = iota
	DigitalInputDevice
	DigitalOutputDevice
	AnalogInputDevice
	AnalogOutputDevice
)

type DeviceDescription struct {
	ID    string
	Kind  DeviceKind
	State interface{}
}

// Device is the interface every kind of device should implement
type Device interface {
	// Unique id for the device
	ID() string

	// Device type
	Kind() DeviceKind

	// Description rapresents the device state plus his type and id at the ime it has been requested
	Description() DeviceDescription
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

// CreateBuses creates the set of available actuators
func CreateBuses() map[string]Bus {
	result := make(map[string]Bus)
	for _, fun := range Actuators {
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
