package actuator

import (
	"iosomething/utils"
)

// Initialize initializes the actuator
var Initialize func()

// Deinitialize is used when actuator is going off
var Deinitialize func()

// ExecuteCommand uses the actuator to run a DigitalCommand
var ExecuteCommand func(command *utils.DigitalCommand)
