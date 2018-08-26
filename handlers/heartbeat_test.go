package handlers

import (
	"testing"
	"time"
)

func TestHeartbeat(t *testing.T) {
	CreateServer(t)
	client, err := CreateClient()
	if err != nil {
		t.Error(err)
	}
	heartbeat := NewHeartBeatHandler(client.id, 200*time.Millisecond)
	sub := client.remote.PubSub.Sub("out")
	go heartbeat.HandlerRoutine(client.remote)

	select {
	case <-sub:
		return

	case <-time.After(300 * time.Millisecond):
		t.Error("Heartbeat not received")
	}
}
