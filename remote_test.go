package chik

import (
	"net"
	"testing"
	"time"

	"github.com/gochik/chik/types"
)

var stopChannel <-chan bool

func testRoutine(t *testing.T, f func(c net.Conn, controller *Controller)) {
	srv, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	s, err := srv.Accept()
	if err != nil {
		t.Fatal(err)
	}
	controller := NewController()
	stopChannel = controller.Connect(s)
	defer controller.Shutdown()

	f(c, controller)
}

func hasStopped() bool {
	for {
		select {
		case <-stopChannel:
			return true

		case <-time.After(1 * time.Second):
			return false
		}
	}
}

func TestInvalidRead(t *testing.T) {
	for _, val := range []struct {
		Name    string
		RawData []byte
	}{
		{"Invalid length", []byte{0, 0, 0, 0, 2, 0, 0}},
		{"Type out of bound", []byte{byte(types.HeartbeatType + 100), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		testRoutine(t, func(c net.Conn, controller *Controller) {
			c.Write(val.RawData)
			if !hasStopped() {
				t.Errorf("%s test failed", val.Name)
			}
		})
	}
}
