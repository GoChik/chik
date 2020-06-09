package chik

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
)

type Message struct {
	sender   uuid.UUID
	receiver uuid.UUID
	command  *types.Command
}

// NewMessage creates a new message
func NewMessage(receiver uuid.UUID, command *types.Command) *Message {
	return &Message{
		sender:   uuid.Nil,
		receiver: receiver,
		command:  command,
	}
}

// ParseMessage handles incoming data and creates a Message object
func ParseMessage(reader io.Reader) (*Message, error) {
	message := Message{}
	var length uint32
	err := binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}

	if length < 16*2 {
		return nil, fmt.Errorf("Message too short, must be at least 32 bytes, got: %d", length)
	}

	err = binary.Read(reader, binary.BigEndian, &message.sender)
	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &message.receiver)
	if err != nil {
		return nil, err
	}

	datalength := length - 16*2

	if datalength > 0 {
		data := make([]byte, datalength)
		err := binary.Read(reader, binary.BigEndian, &data)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &message.command)
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
	return m.command
}

// Bytes returns the binary rapresentation of the message
func (m *Message) Bytes() ([]byte, error) {
	data, err := json.Marshal(m.command)
	if err != nil {
		return []byte{}, err
	}
	length := uint32(16*2 + len(data))
	buffer := bytes.Buffer{}

	binary.Write(&buffer, binary.BigEndian, length)
	binary.Write(&buffer, binary.BigEndian, m.sender)
	binary.Write(&buffer, binary.BigEndian, m.receiver)
	binary.Write(&buffer, binary.BigEndian, data)

	return buffer.Bytes(), nil
}

// Equal compares two messages
func Equal(v1, v2 *Message) bool {
	if v1 != nil && v2 != nil {
		if v1.sender != v2.sender {
			return false
		}

		if v1.receiver != v2.receiver {
			return false
		}

		if !reflect.DeepEqual(v1.command, v2.command) {
			return false
		}

		return true
	}

	return v1 == v2
}

func (m Message) String() string {
	return fmt.Sprintf("{%v, %v, %v}", m.sender, m.receiver, m.command)
}
