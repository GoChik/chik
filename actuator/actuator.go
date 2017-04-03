package actuator

// Actuator interface
type Actuator interface {
	// Initialize initializes the actuator
	Initialize()

	// Deinitialize is used when actuator is going off
	Deinitialize()

	// Execute uses the actuator to execute the action specified
	// on the data passed to it, returns the reply
	Execute(data []byte) []byte
}

// NewActuator creates a new actuator
var NewActuator = func() Actuator {
	return nil
}
