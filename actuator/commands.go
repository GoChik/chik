package actuator

type CommandType uint8

const (
	PUSH_BUTTON CommandType = iota
	SWITCH_ON
	SWITCH_OFF
	TOGGLE_ON_OFF
	GET_STATUS
)

// DigitalCommand used to instruct the appliance to execute
// an action on a gpio pin
type DigitalCommand struct {
	Pin     int         `json:",string"`
	Command CommandType `json:",string"`
}

// StatusIndication is the response to a status request Command
// Pin indicates the pin which status refers to
// Status is the current value
type StatusIndication struct {
	Pin    int
	Status bool
}
