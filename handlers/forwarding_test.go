package handlers

import (
	"iosomething"
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

	forwarded := client1.remote.PubSub.Sub(iosomething.SimpleCommandType.String())
	client1.remote.PubSub.Pub(iosomething.NewMessage(iosomething.HeartbeatType, client1.id, uuid.Nil, []byte("")), "out")
	client2.remote.PubSub.Pub(iosomething.NewMessage(iosomething.SimpleCommandType, client2.id, client1.id, []byte("Hello")), "out")

	select {
	case <-forwarded:
		logrus.Debug("forwarding done")

	case <-time.After(100 * time.Millisecond):
		t.Fail()
	}
}
