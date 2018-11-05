//go:generate stringer -type=CommandType

package chik

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
)

// CommandType is an enum of all the possible commands
type Action uint16

// EnabledDays days on which a TimedCommand is enabled
// used as binary flag
type EnabledDays uint16

// MsgType represent the type of the current message
type CommandType uint8

// Message types used in various plugins
const (
	HeartbeatType CommandType = iota
	DigitalCommandType
	TimerCommandType
	StatusRequestCommandType
	StatusReplyCommandType
	VersionRequestCommandType
	VersionReplyCommandType
	SunsetCommandType

	messageBound
)

// Available command types
const (
	SET     Action = iota // Turn on/activate something
	RESET                 // Turn off/deactivate something
	TOGGLE                // Toggle something from on/activated to off/deactivated
	PUSH                  // Used to define actions shuch as the one of pushing a button
	GET                   // Retrieve a value
	SUNSET                // Sunset related command
	SUNRISE               // Sunrise Related command
	DELETE                // Delete a value
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

// Command is the root object in every message
type Command struct {
	Type CommandType
	Data json.RawMessage
}

// NewCommand creates a command given a type and his content
func NewCommand(t CommandType, data interface{}) *Command {
	body, err := json.Marshal(data)
	if err != nil {
		logrus.Error("Cannot compose command: ", err)
		return nil
	}

	return &Command{t, json.RawMessage(body)}
}

// SimpleCommand is used to send a basic request regarding the whole system
type SimpleCommand struct {
	Command JSIntArr `json:",JSIntArr"`
}

// DigitalCommand used to instruct the appliance to execute
// a Command on a gpio pin
type DigitalCommand struct {
	Command JSIntArr `json:",JSIntArr"`
	Pin     int      `json:",int,omitempty"`
}

// TimedCommand represent a command with an associated delay in minutes
// if TimerID is zero it means it is a new timer. otherwise it should edit the timer with
// the corresponding id
type TimedCommand struct {
	Action  JSIntArr    `json:",JSIntArr"`
	TimerID uint16      `json:",int,omitempty"`
	Time    JSONTime    `json:",string,omitempty"`
	Repeat  EnabledDays `json:",int,omitempty"`
	Command *Command
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
