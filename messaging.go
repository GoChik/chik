package chik

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
)

type Message struct {
	length   uint32
	sender   uuid.UUID
	receiver uuid.UUID
	data     []byte
}

// NewMessage creates a new message
func NewMessage(receiver uuid.UUID, command *types.Command) *Message {
	data, err := json.Marshal(command)
	if err != nil {
		logrus.Error("Failed to create a message, parsing data failed")
		return nil
	}
	messageLen := (16 * 2) + uint32(len(data))

	return &Message{
		length:   messageLen,
		sender:   uuid.Nil,
		receiver: receiver,
		data:     data,
	}
}

// ParseMessage handles incoming data and creates a Message object
func ParseMessage(reader io.Reader) (*Message, error) {
	message := Message{}

	err := binary.Read(reader, binary.BigEndian, &message.length)
	if err != nil {
		return nil, err
	}

	if message.length < 16*2 {
		return nil, fmt.Errorf("Message too short, must be at least 32 bytes, got: %d", message.length)
	}

	err = binary.Read(reader, binary.BigEndian, &message.sender)
	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &message.receiver)
	if err != nil {
		return nil, err
	}

	datalength := message.length - 16*2

	if datalength > 0 {
		message.data = make([]byte, datalength)
		err := binary.Read(reader, binary.BigEndian, &message.data)
		if err != nil {
			return nil, err
		}
	}

	return &message, nil
}

// SenderUUID returns the sender identity
func (m *Message) SenderUUID() uuid.UUID {
	return m.sender
}

// ReceiverUUID returns the receiver identity
func (m *Message) ReceiverUUID() (uuid.UUID, error) {
	return m.receiver, nil
}

// Command returns message content as a Command object
func (m *Message) Command() *types.Command {
	var command types.Command
	err := json.Unmarshal(m.data, &command)
	if err != nil {
		return nil
	}
	return &command
}

// Bytes returns the binary rapresentation of the message
func (m *Message) Bytes() []byte {
	buffer := bytes.Buffer{}

	binary.Write(&buffer, binary.BigEndian, m.length)
	binary.Write(&buffer, binary.BigEndian, m.sender)
	binary.Write(&buffer, binary.BigEndian, m.receiver)
	binary.Write(&buffer, binary.BigEndian, m.data)

	return buffer.Bytes()
}

// Equal compares two messages
func Equal(v1, v2 *Message) bool {
	if v1.length != v2.length {
		return false
	}

	if v1.sender != v2.sender {
		return false
	}

	if v1.receiver != v2.receiver {
		return false
	}

	if !bytes.Equal(v1.data, v2.data) {
		return false
	}

	return true
}

func (m Message) String() string {
	return fmt.Sprintf("{%v, %v, %s}", m.sender, m.receiver, m.data)
}
