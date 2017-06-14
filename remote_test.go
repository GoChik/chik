package iosomething

import (
	"net"
	"testing"
	"time"

	"bytes"

	uuid "github.com/satori/go.uuid"
)

func testRoutine(t *testing.T, f func(c net.Conn, r *Remote)) {
	srv, _ := net.Listen("tcp", "127.0.0.1:8080")
	c, _ := net.Dial("tcp", "127.0.0.1:8080")
	s, _ := srv.Accept()
	r := NewRemote(s, 100*time.Millisecond)

	f(c, r)

	srv.Close()
}

func hasStopped(stop chan bool) bool {
	select {
	case <-stop:
		return true

	case <-time.After(500 * time.Millisecond):
		return false
	}
}

func TestInvalidRead(t *testing.T) {
	for _, val := range []struct {
		Name    string
		RawData []byte
	}{
		{"Empty message", []byte{0}},
		{"Invalid length", []byte{0, 0, 0, 0, 2, 0, 0}},
		{"Type out of bound", []byte{byte(MessageBound), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		testRoutine(t, func(c net.Conn, r *Remote) {
			stop := r.StopChannel()
			c.Write(val.RawData)
			if !hasStopped(stop) {
				t.Errorf("%s test failed", val.Name)
			}
		})
	}
}

func TestTerminateTwice(t *testing.T) {
	testRoutine(t, func(c net.Conn, r *Remote) {
		stop := r.StopChannel()
		r.Terminate()
		if !hasStopped(stop) {
			t.Error("Expecting a stop signal")
		}
		r.Terminate()
		if !hasStopped(stop) {
			t.Error("Expecting a stop signal")
		}

		r.OutBuffer <- NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte{})
		if hasStopped(stop) {
			t.Error("Write routine still running, Terminate called again")
		}
	})
}

func TestRead(t *testing.T) {
	for _, val := range [][]byte{
		[]byte{byte(MESSAGE), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{byte(HEARTBEAT), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		testRoutine(t, func(c net.Conn, r *Remote) {
			c.Write(val)
			message := <-r.InBuffer
			if !bytes.Equal(val, message.Bytes()) {
				t.Errorf("Unexpected message decoded: \nexpecting: %v\ngot:       %v", val, message.Bytes())
			}
		})
	}
}

func TestWrite(t *testing.T) {
	for _, val := range []*Message{
		NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte{}),
		NewMessage(HEARTBEAT, uuid.Nil, uuid.Nil, []byte{}),
		NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte("HELLO WORLD")),
	} {
		testRoutine(t, func(c net.Conn, r *Remote) {
			expected := val.Bytes()
			r.OutBuffer <- val
			readed := make([]byte, len(expected))
			c.Read(readed)
			if !bytes.Equal(readed, expected) {
				t.Errorf("Unexpected message decoded: \nexpecting: %v\ngot:       %v", expected, readed)
			}
		})
	}
}
