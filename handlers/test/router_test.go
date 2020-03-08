package test

import (
	"testing"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func TestForwarding(t *testing.T) {
	CreateServer(t)

	client1, err := CreateClient()
	if err != nil {
		t.Fatal(err)
	}

	client2, err := CreateClient()
	if err != nil {
		t.Fatal(err)
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Sender:", client1.id, "receiver:", client2.id)

	forwarded := client1.remote.Sub(types.DigitalCommandType.String())
	time.Sleep(500 * time.Millisecond) // TODO: fix the handshake
	client1.remote.PubMessage(chik.NewMessage(uuid.Nil, types.NewCommand(types.HeartbeatType, nil)), types.AnyOutgoingCommandType.String())
	client2.remote.PubMessage(chik.NewMessage(uuid.Nil, types.NewCommand(types.HeartbeatType, nil)), types.AnyOutgoingCommandType.String())
	time.Sleep(500 * time.Millisecond) // TODO: fix the handshake
	client2.remote.Pub(types.NewCommand(types.DigitalCommandType, types.SimpleCommand{}), client1.id)

	select {
	case <-forwarded:
		logrus.Debug("forwarding done")

	case <-time.After(1000 * time.Millisecond):
		t.Fail()
	}
}
