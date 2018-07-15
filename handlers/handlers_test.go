package handlers

import (
	"iosomething"
	"net"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
)

type TestClient struct {
	remote *iosomething.Remote
	id     uuid.UUID
}

func CreateServer(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:4000")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			srv := iosomething.NewRemote(conn, 10*time.Millisecond)
			go NewForwardingHandler(peers).HandlerRoutine(srv)
		}
	}()
	return listener
}

func CreateClient() (client TestClient, err error) {
	conn, err := net.Dial("tcp", "127.0.0.1:4000")
	if err != nil {
		return
	}

	client = TestClient{iosomething.NewRemote(conn, 10*time.Millisecond), uuid.NewV1()}
	return
}
