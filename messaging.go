//go:generate stringer -type=MsgType

package iosomething

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/satori/go.uuid"
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

	if message.header.MsgType >= messageBound {
		return nil, fmt.Errorf("Message type out of bound %v", message.header.MsgType)
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
	return m.sender, nil
}

// ReceiverUUID returns the receiver identity
func (m *Message) ReceiverUUID() (uuid.UUID, error) {
	if m.header.MsgType == HeartbeatType {
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

// Equal compares two messages
func Equal(v1, v2 *Message) bool {
	if v1.header != v2.header {
		return false
	}

	if !uuid.Equal(v1.sender, v2.sender) {
		return false
	}

	if !uuid.Equal(v1.receiver, v2.receiver) {
		return false
	}

	if !bytes.Equal(v1.data, v2.data) {
		return false
	}

	return true
}
