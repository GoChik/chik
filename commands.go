package iosomething

type CommandType uint8

const (
	PUSH_BUTTON   CommandType = iota // on followed by off command
	SWITCH_ON                        // Turn on something
	SWITCH_OFF                       // Turn off something
	TOGGLE_ON_OFF                    // Toggle somethinf
	GET_STATUS                       // Get current status of an appliance
	GET_VERSION                      // Get version and available updates
	DO_UPDATE                        // Instruct the client to update himself
)

// SimpleCommand is used to send a basic request regarding the whole system
type SimpleCommand struct {
	Command CommandType `json:",int"`
}

// TimedCommand represent a command with an associated delay in minutes
// if TimerID is zero it means id has not been set
type TimedCommand struct {
	TimerID      uint16      `json:",int"`
	Command      CommandType `json:",int"`
	DelayMinutes int         `json:",int"`
}

// DigitalCommand used to instruct the appliance to execute
// a TimedCommand on a gpio pin
type DigitalCommand struct {
	TimedCommand
	Pin int `json:",int"`
}

// StatusIndication is the response to a status request Command
// Pin indicates the pin which status refers to
// Status is the current value
type StatusIndication struct {
	Pin    int  `json:",int"`
	Status bool `json:",bool"`
	Timers []TimedCommand
}

// AIInfoMessage sent from the android application to inform
// when user came home and what appliances are enabled
type AIInfoMessage struct {
	EnabledPins []int
	AtHome      bool
}

// VersionIndication returns info about the current version and the optional update available
type VersionIndication struct {
	CurrentVersion   string `json:",string"`
	AvailableVersion string `json:",string"`
}
