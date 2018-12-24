package handlers

import (
	"testing"
	"time"

	"github.com/gochik/chik"
)

func TestHeartbeat(t *testing.T) {
	client, err := CreateClient()
	if err != nil {
		t.Error(err)
	}
	heartbeat := NewHeartBeatHandler(200 * time.Millisecond)
	sub := client.remote.Sub(chik.OutgoingMessage)
	go heartbeat.Run(client.remote)

	select {
	case <-sub:
		return

	case <-time.After(300 * time.Millisecond):
		t.Error("Heartbeat not received")
	}
}
