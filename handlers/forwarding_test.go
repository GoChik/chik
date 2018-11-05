package handlers

import (
	"chik"
	"sync"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

var peers = sync.Map{}

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
	logrus.Debug("Sender:", client1.id, "receiver:", client2.id)

	forwarded := client1.remote.PubSub.Sub(chik.DigitalCommandType.String())
	client1.remote.PubSub.Pub(chik.NewMessage(uuid.Nil, chik.NewCommand(chik.HeartbeatType, nil)), "out")
	time.Sleep(100 * time.Millisecond) // TODO: fix the handshake
	client2.remote.PubSub.Pub(chik.NewMessage(client1.id, chik.NewCommand(chik.DigitalCommandType, nil)), "out")

	select {
	case <-forwarded:
		logrus.Debug("forwarding done")

	case <-time.After(100 * time.Millisecond):
		t.Fail()
	}
}
