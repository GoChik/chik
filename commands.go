package iosomething

type CommandType uint8

const (
	PUSH_BUTTON CommandType = iota
	SWITCH_ON
	SWITCH_OFF
	TOGGLE_ON_OFF
	GET_STATUS
)

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

type AIConfigurationMessage struct {
	timeSpan        int
	collectInterval int
}
