//go:generate stringer -type=CommandType

package types

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

// EnabledDays days on which a TimedCommand is enabled
// used as binary flag
type EnabledDays uint16

// CommandType represent the type of the current message
type CommandType uint8

// Command types types used in various plugins
const (
	HeartbeatType CommandType = iota
	DigitalCommandType
	AnalogCommandType
	StatusCommandType
	StatusNotificationCommandType
	VersionRequestCommandType
	VersionReplyCommandType

	// Actor commands: request and reply
	ActionRequestCommandType
	ActionReplyCommandType

	// private commands (sent on the loopback address)
	StatusUpdateCommandType
	NullCommandType

	// Telegram notifications
	TelegramNotificationCommandType

	// special command types
	AnyIncomingCommandType
	AnyOutgoingCommandType

	messageBound
)

// Action is an enum of all the possible commands
type Action uint16

// Available actions
const (
	SET    Action = iota // Turn on/activate something
	RESET                // Turn off/deactivate something
	TOGGLE               // Toggle something from on/activated to off/deactivated
	PUSH                 // Used to define actions shuch as the one of pushing a button
	GET                  // Retrieve a value
)

// AnalogValueType is an enum to define how to handle an analog value
type AnalogValueType uint8

// Available AnalogValueType
const (
	Absolute AnalogValueType = iota
	Relative
)

// Command is the root object in every message
type Command struct {
	Type CommandType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (c *Command) String() string {
	return fmt.Sprintf("{type: %v, data: %s}", c.Type, string(c.Data))
}

// NewCommand creates a command given a type and his content
func NewCommand(t CommandType, data interface{}) *Command {
	body, err := json.Marshal(data)
	if err != nil {
		log.Error().Msgf("Cannot compose command: %v", err)
		return nil
	}

	return &Command{t, json.RawMessage(body)}
}

// SimpleCommand is used to send a basic request regarding the whole system
type SimpleCommand struct {
	Action Action `json:"action,int"`
}

// DigitalCommand used to instruct the appliance to execute
// a Command on a gpio pin
type DigitalCommand struct {
	Action      Action `json:"action,int"`
	ApplianceID string `json:"applianceID"`
}

// AnalogCommand is a command to set a value on an analog device.
// By default the ValueType is 0: Absolute.
type AnalogCommand struct {
	ApplianceID string          `json:"applianceID"`
	Value       float64         `json:"value,float"`
	ValueType   AnalogValueType `json:"value_type,int,omitempty"`
}

// Status is the response to a status request Command
// the key is the handler name
// value can be anything
type Status map[string]interface{}

// VersionIndication returns info about the current version and the optional update available
type VersionIndication struct {
	CurrentVersion   string
	AvailableVersion string
}
