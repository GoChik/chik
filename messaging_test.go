package chik

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"bytes"

	uuid "github.com/gofrs/uuid"
)

type TestData struct {
	Raw    []byte
	Packed *Message
}

var data = []TestData{
	{
		[]byte{0, 0, 0, 0x24, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6E, 0x75, 0x6C, 0x6C},
		NewMessage(uuid.Nil, nil),
	},
	{
		[]byte{0, 0, 0, 0x36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x7B, 0x22, 0x54, 0x79, 0x70, 0x65, 0x22, 0x3A, 0x30, 0x2C, 0x22, 0x44, 0x61, 0x74, 0x61, 0x22, 0x3A, 0x6E, 0x75, 0x6C, 0x6C, 0x7D},
		NewMessage(uuid.Nil, NewCommand(HeartbeatType, nil)),
	},
}

func TestEncode(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stderr)
	for _, val := range data {
		reader := bytes.NewReader(val.Raw)
		msg, err := ParseMessage(reader)
		if err != nil {
			t.Error("Test failed with error: ", err)
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
