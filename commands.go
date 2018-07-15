//go:generate stringer -type=MsgType

package iosomething

// CommandType is an enum of all the possible commands
type CommandType uint16

// EnabledDays days on which a TimedCommand is enabled
// used as binary flag
type EnabledDays uint16

// MsgType represent the type of the current message
type MsgType uint8

// Message types mapped 1-1 with underneath structs
const (
	SimpleCommandType MsgType = iota
	DigitalCommandType
	TimedCommandType
	StatusIndicationType
	VersionIndicationType
	HeartbeatType

	messageBound
)

// Available command types
const (
	PUSH_BUTTON   CommandType = iota // on followed by off command
	SWITCH_ON                        // Turn on something
	SWITCH_OFF                       // Turn off something
	TOGGLE_ON_OFF                    // Toggle something
	GET_STATUS                       // Get current status
	GET_VERSION                      // Get version and available updates
	DO_UPDATE                        // Instruct the client to update himself
	DELETE_TIMER                     // remove a timer
	HEARTBEAT                        // Empty heartbeat message
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

// DigitalCommand used to instruct the appliance to execute
// a Command on a gpio pin
type DigitalCommand struct {
	Command CommandType `json:",int"`
	Pin     int         `json:",int,omitempty"`
}

// TimedCommand represent a command with an associated delay in minutes
// if TimerID is zero it means it is a new timer. otherwise it should edit the timer with
// the corresponding id
type TimedCommand struct {
	DigitalCommand
	TimerID uint16      `json:",int,omitempty"`
	Time    JSONTime    `json:",string,omitempty"`
	Repeat  EnabledDays `json:",int,omitempty"`
}

// Status is the response to a status request Command
// Pin indicates the pin which status refers to
// Status is the current value
type Status map[string]interface{}

// VersionIndication returns info about the current version and the optional update available
type VersionIndication struct {
	CurrentVersion   string
	AvailableVersion string
}
