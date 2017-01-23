package actuator

import (
	"iosomething/utils"
)

// Actuator interface
type Actuator interface {
	// Initialize initializes the actuator
	Initialize()
	// Deinitialize is used when actuator is going off
	Deinitialize()
	// ExecuteCommand uses the actuator to run a DigitalCommand
	ExecuteCommand(command *utils.DigitalCommand)
}

// NewActuator creates a new actuator
var NewActuator = func() Actuator {
	return nil
}
