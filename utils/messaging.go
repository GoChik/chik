//go:generate stringer -type=MsgType
//go:generate stringer -type=CommandType

package utils

import (
	"bufio"
	"encoding/binary"
)

// MsgType represent the type of the current message
type MsgType uint8

// IDENTITY: Publish my identity
// MESSAGE: Message with receiver and content
// HEARTBEAT: Empty message
const (
	IDENTITY MsgType = iota
	MESSAGE
	HEARTBEAT
)

type CommandType uint8

const (
	PUSH_BUTTON CommandType = iota
	SWITCH_ON
	SWITCH_OFF
	TOGGLE_ON_OFF
)

type MsgHeader struct {
	MsgType MsgType
	MsgLen  uint32
}

type DigitalCommand struct {
	Pin     int         `json:",string"`
	Command CommandType `json:",string"`
}

// ParseHeader creates MsgHeader from raw data
// message header: | type 1B | length 4B |
func ParseHeader(reader *bufio.Reader) (MsgHeader, error) {
	header := MsgHeader{}
	err := binary.Read(reader, binary.BigEndian, &header)

	return header, err
}
