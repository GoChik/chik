package handlers

import (
	"chik"
	"net"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

type TestClient struct {
	remote *chik.Remote
	id     uuid.UUID
}

var address net.Addr

func CreateServer(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	address = listener.Addr()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			id, err := uuid.NewV1()
			if err != nil {
				t.Fatal(err)
			}
			srv := chik.NewRemote(id, conn, 10*time.Millisecond)
			go NewForwardingHandler(&peers).Run(srv)
		}
	}()
	return listener
}

func CreateClient() (client TestClient, err error) {
	conn, err := net.Dial("tcp", address.String())
	if err != nil {
		return
	}

	id, _ := uuid.NewV1()
	client = TestClient{chik.NewRemote(id, conn, 10*time.Millisecond), id}
	return
}