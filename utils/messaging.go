//go:generate stringer -type=MsgType
//go:generate stringer -type=CommandType

package utils

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/satori/go.uuid"
)

// MsgType represent the type of the current message
type MsgType uint8

// MESSAGE: Message with receiver and content
// HEARTBEAT: Empty message
const (
	MESSAGE MsgType = iota
	HEARTBEAT
)

type msgHeader struct {
	MsgType MsgType
	MsgLen  uint32
}

type Message struct {
	header   msgHeader
	sender   uuid.UUID
	receiver uuid.UUID
	data     []byte
}

type CommandType uint8

const (
	PUSH_BUTTON CommandType = iota
	SWITCH_ON
	SWITCH_OFF
	TOGGLE_ON_OFF
)

type DigitalCommand struct {
	Pin     int         `json:",string"`
	Command CommandType `json:",string"`
}

// NewMessage creates a new message
func NewMessage(msgtype MsgType, sender uuid.UUID, receiver uuid.UUID, data []byte) *Message {
	return &Message{
		header:   msgHeader{MsgType: msgtype, MsgLen: 16*2 + uint32(len(data))},
		sender:   sender,
		receiver: receiver,
		data:     data,
	}
}

// ParseMessage handles incoming data and creates a Message object
func ParseMessage(reader *bufio.Reader) (*Message, error) {
	message := Message{}

	err := binary.Read(reader, binary.BigEndian, &message.header)
	if err != nil {
		return nil, err
	}

	if message.header.MsgType != HEARTBEAT {
		err := binary.Read(reader, binary.BigEndian, &message.sender)
		if err != nil {
			return nil, err
		}

		err = binary.Read(reader, binary.BigEndian, &message.receiver)
		if err != nil {
			return nil, err
		}
	}

	datalength := message.header.MsgLen - 16*2 // 16 is the uuid size
	if datalength > 0 {
		message.data = make([]byte, datalength)
		err := binary.Read(reader, binary.BigEndian, &message.data)
		if err != nil {
			return nil, err
		}
	}

	return &message, nil
}

// Type returns the message type
func (m *Message) Type() MsgType {
	return m.header.MsgType
}

// SenderUUID returns the sender identity
func (m *Message) SenderUUID() (uuid.UUID, error) {
	if m.header.MsgType == HEARTBEAT {
		return uuid.Nil, errors.New("HEARTBEAT message does not have a sender")
	}

	return m.sender, nil
}

// ReceiverUUID returns the receiver identity
func (m *Message) ReceiverUUID() (uuid.UUID, error) {
	if m.header.MsgType == HEARTBEAT {
		return uuid.Nil, errors.New("HEARTBEAT message does not have a receiver")
	}

	return m.receiver, nil
}

// Data returns message content
func (m *Message) Data() []byte {
	return m.data
}

// Bytes returns the binary rapresentation of the message
func (m *Message) Bytes() []byte {
	buffer := new(bytes.Buffer)

	binary.Write(buffer, binary.BigEndian, m.header)

	if m.header.MsgType == HEARTBEAT {
		return buffer.Bytes()
	}

	binary.Write(buffer, binary.BigEndian, m.sender)
	binary.Write(buffer, binary.BigEndian, m.receiver)
	binary.Write(buffer, binary.BigEndian, m.data)

	return buffer.Bytes()
}
