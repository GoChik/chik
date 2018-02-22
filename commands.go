package iosomething

// CommandType is an enum of all the possible commands
type CommandType uint8

// EnabledDays days on which a TimedCommand is enabled
// used as binary flag
type EnabledDays uint16

// Available command types
const (
	PUSH_BUTTON   CommandType = iota // on followed by off command
	SWITCH_ON                        // Turn on something
	SWITCH_OFF                       // Turn off something
	TOGGLE_ON_OFF                    // Toggle somethinf
	GET_STATUS                       // Get current status of an appliance
	GET_VERSION                      // Get version and available updates
	DO_UPDATE                        // Instruct the client to update himself
	DELETE_TIMER                     // remove a timer
)

// Days of the week
const (
	Noday     EnabledDays = 0x00
	Sunday    EnabledDays = 0x01
	Monday    EnabledDays = 0x02
	Tuesday   EnabledDays = 0x04
	Wednesday EnabledDays = 0x08
	Thursday  EnabledDays = 0x10
	Friday    EnabledDays = 0x20
	Saturday  EnabledDays = 0x40
)

// SimpleCommand is used to send a basic request regarding the whole system
type SimpleCommand struct {
	Command CommandType `json:",int"`
}

// TimedCommand represent a command with an associated delay in minutes
// if TimerID is zero it means it is a new timer. otherwise it should edit the timer with
// the corresponding id
type TimedCommand struct {
	TimerID uint16      `json:",int,omitempty"`
	Command CommandType `json:",int"`
	Time    JSONTime    `json:",string,omitempty"`
	Repeat  EnabledDays `json:",int,omitempty"`
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

// VersionIndication returns info about the current version and the optional update available
type VersionIndication struct {
	CurrentVersion   string
	AvailableVersion string
}
