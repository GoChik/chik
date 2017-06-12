package actuator

// Actuator interface
type Actuator interface {
	// Initialize initializes the actuator
	Initialize()

	// Deinitialize is used when actuator is going off
	Deinitialize()

	// TurnOn turns the specified pin on
	TurnOn(pin int)

	// TurnOff turns the specified pin off
	TurnOff(pin int)

	// getStatus returns wether the pin is on (true) or off (false)
	GetStatus(pin int) bool
}

// NewActuator creates a new actuator
var NewActuator = func() Actuator {
	return nil
}
