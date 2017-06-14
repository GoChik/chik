package iosomething

import (
	"testing"

	"bytes"

	uuid "github.com/satori/go.uuid"
)

type TestData struct {
	Raw    []byte
	Packed *Message
}

var data = []TestData{
	{
		[]byte{0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte{}),
	},
	{
		[]byte{1, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		NewMessage(HEARTBEAT, uuid.Nil, uuid.Nil, []byte{}),
	},
	{
		[]byte{0, 0, 0, 0, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4},
		NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte{1, 2, 3, 4}),
	},
}

func TestEncode(t *testing.T) {
	for _, val := range data {
		reader := bytes.NewReader(val.Raw)
		msg, err := ParseMessage(reader)
		if err != nil {
			t.Error(err)
		}
		if !Equal(msg, val.Packed) {
			t.Errorf("Encode failed:\nexpected: %v\nactual:   %v", val.Packed, msg)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, val := range data {
		b := val.Packed.Bytes()
		if !bytes.Equal(b, val.Raw) {
			t.Errorf("Encode failed:\nexpected: %v\nactual:   %v", val.Raw, b)
		}
	}
}
