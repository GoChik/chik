package chik

import (
	"context"
	"net"
	"testing"
	"time"
)

var ctx context.Context

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
	var cancel context.CancelFunc
	ctx, cancel = StartRemote(controller, s, MaxIdleTime)
	defer cancel()

	f(c, controller)
}

func hasStopped() bool {
	for {
		select {
		case <-ctx.Done():
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
	} {
		testRoutine(t, func(c net.Conn, controller *Controller) {
			c.Write(val.RawData)
			if !hasStopped() {
				t.Errorf("%s test failed", val.Name)
			}
		})
	}
}
