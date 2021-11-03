package test

import (
	"testing"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
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

	forwarded := client1.remote.Sub(types.DigitalCommandType.String())
	client2.remote.PubMessage(chik.NewMessage(getServerID(), types.NewCommand(types.HeartbeatType, nil)), types.AnyOutgoingCommandType.String())
	time.Sleep(100 * time.Millisecond)
	client1.remote.PubMessage(chik.NewMessage(getServerID(), types.NewCommand(types.HeartbeatType, nil)), types.AnyOutgoingCommandType.String())
	time.Sleep(100 * time.Millisecond)
	client2.remote.Pub(types.NewCommand(types.DigitalCommandType, types.SimpleCommand{}), client1.id)

	select {
	case <-forwarded:
		t.Log("OK")

	case <-time.After(1000 * time.Millisecond):
		t.Fail()
	}
}
