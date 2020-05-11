//go:generate stringer -type=CommandType

package types

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

// Action is an enum of all the possible commands
type Action uint16

// EnabledDays days on which a TimedCommand is enabled
// used as binary flag
type EnabledDays uint16

// CommandType represent the type of the current message
type CommandType uint8

// Command types types used in various plugins
const (
	HeartbeatType CommandType = iota
	DigitalCommandType
	StatusSubscriptionCommandType
	StatusNotificationCommandType
	VersionRequestCommandType
	VersionReplyCommandType

	// private commands (sent on the loopback address)
	StatusUpdateCommandType
	IODeviceStatusChangedCommandType

	// special command types
	AnyIncomingCommandType
	AnyOutgoingCommandType

	messageBound
)

// Available command types
const (
	SET    Action = iota // Turn on/activate something
	RESET                // Turn off/deactivate something
	TOGGLE               // Toggle something from on/activated to off/deactivated
	PUSH                 // Used to define actions shuch as the one of pushing a button
	GET                  // Retrieve a value
)

// Command is the root object in every message
type Command struct {
	Type CommandType
	Data json.RawMessage
}

func (c *Command) String() string {
	return fmt.Sprintf("{Type: %v, Data: %s}", c.Type, string(c.Data))
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
	Action Action `json:"action,int"`
}

// DigitalCommand used to instruct the appliance to execute
// a Command on a gpio pin
type DigitalCommand struct {
	Action      Action `json:"action,int"`
	ApplianceID string
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
