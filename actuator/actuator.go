package actuator

import (
	"fmt"
)

var actuators []func() Actuator

// DigitalDevice is the interface that a device must implement
type DigitalDevice interface {
	ID() string
	TurnOn()
	TurnOff()
	Toggle()
	GetStatus() bool
}

// CreateActuators creates the set of available actuators
func CreateActuators() map[string]Actuator {
	result := make(map[string]Actuator)
	for _, fun := range actuators {
		a := fun()
		result[a.String()] = a
	}
	return result
}

// Actuator interface
type Actuator interface {
	fmt.Stringer
	// Initialize initializes the actuator
	Initialize(config interface{})

	// Deinitialize is used when actuator is going off
	Deinitialize()

	Device(id string) DigitalDevice
	DeviceIds() []string
}
