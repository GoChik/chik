//go:generate stringer -type=MsgType
//go:generate stringer -type=CommandType

package iosomething

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

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

// NewMessage creates a new message
func NewMessage(msgtype MsgType, sender uuid.UUID, receiver uuid.UUID, data []byte) *Message {
	messageLen := (16 * 2) + uint32(len(data))

	return &Message{
		header:   msgHeader{MsgType: msgtype, MsgLen: messageLen},
		sender:   sender,
		receiver: receiver,
		data:     data,
	}
}

// ParseMessage handles incoming data and creates a Message object
func ParseMessage(reader io.Reader) (*Message, error) {
	message := Message{}

	err := binary.Read(reader, binary.BigEndian, &message.header)
	if err != nil {
		return nil, err
	}

	datalength := message.header.MsgLen

	if datalength < 16*2 {
		return nil, fmt.Errorf("Message too short, must be at least 32 bytes, got: %d", datalength)
	}

	err = binary.Read(reader, binary.BigEndian, &message.sender)
	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &message.receiver)
	if err != nil {
		return nil, err
	}

	datalength -= 16 * 2

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
	binary.Write(buffer, binary.BigEndian, m.sender)
	binary.Write(buffer, binary.BigEndian, m.receiver)
	binary.Write(buffer, binary.BigEndian, m.data)

	return buffer.Bytes()
}
