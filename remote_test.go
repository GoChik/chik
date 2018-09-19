package chik

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

var testSub chan interface{}

func testRoutine(t *testing.T, f func(c net.Conn, r *Remote)) {
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
	id, err := uuid.NewV1()
	if err != nil {
		t.Fatal(err)
	}
	r := NewRemote(id, s, 500*time.Millisecond)
	defer r.Terminate()
	testSub = r.PubSub.Sub("test")
	r.PubSub.Pub("echo", "test")

	f(c, r)
}

func hasStopped(remote *Remote) bool {
	for {
		select {
		case _, more := <-testSub:
			if !more {
				return true
			}
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
		{"Empty message", []byte{0}},
		{"Invalid length", []byte{0, 0, 0, 0, 2, 0, 0}},
		{"Type out of bound", []byte{byte(SimpleCommandType + 100), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		testRoutine(t, func(c net.Conn, r *Remote) {
			c.Write(val.RawData)
			if !hasStopped(r) {
				t.Errorf("%s test failed", val.Name)
			}
		})
	}
}

func TestTerminateTwice(t *testing.T) {
	testRoutine(t, func(c net.Conn, r *Remote) {
		r.Terminate()
		if !hasStopped(r) {
			t.Error("Expecting a stop signal")
		}
		r.Terminate()
		if !hasStopped(r) {
			t.Error("Expecting a stop signal")
		}
	})
}

func TestRead(t *testing.T) {
	for _, val := range [][]byte{
		[]byte{byte(SimpleCommandType), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		[]byte{byte(SimpleCommandType), 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		testRoutine(t, func(c net.Conn, r *Remote) {
			c.Write(val)
			messageRaw := <-r.PubSub.Sub(SimpleCommandType.String())
			message := messageRaw.(*Message)
			if !bytes.Equal(val, message.Bytes()) {
				t.Errorf("Unexpected message decoded: \nexpecting: %v\ngot:       %v", val, message.Bytes())
			}
		})
	}
}

// TODO: fix this one
// func TestWrite(t *testing.T) {
// 	for _, val := range []*Message{
// 		NewMessage(SimpleCommandType, uuid.Nil, uuid.Nil, []byte("{}")),
// 		NewMessage(HeartbeatType, uuid.Nil, uuid.Nil, []byte("")),
// 		NewMessage(SimpleCommandType, uuid.Nil, uuid.Nil, []byte("{\"hello_world\": true}")),
// 	} {
// 		testRoutine(t, func(c net.Conn, r *Remote) {
// 			expected := val.Bytes()
// 			r.PubSub.Pub(val, "out")
// 			readed := make([]byte, len(expected))
// 			err := binary.Read(c, binary.BigEndian, readed)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			if !bytes.Equal(readed, expected) {
// 				t.Errorf("Unexpected message decoded: \nexpecting: %v\ngot:       %v", expected, readed)
// 			}
// 		})
// 	}
// }
