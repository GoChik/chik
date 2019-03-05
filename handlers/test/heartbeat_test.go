package test

import (
	"testing"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/handlers/heartbeat"
)

func TestHeartbeat(t *testing.T) {
	client, err := CreateClient()
	if err != nil {
		t.Error(err)
	}
	heartbeat := heartbeat.New(200 * time.Millisecond)
	sub := client.remote.Sub(chik.OutgoingMessage)
	go heartbeat.Run(client.remote)

	select {
	case <-sub:
		return

	case <-time.After(300 * time.Millisecond):
		t.Error("Heartbeat not received")
	}
}
