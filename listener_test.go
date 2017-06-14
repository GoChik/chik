package iosomething

import (
	"net"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
)

type fakeHandler struct {
	status chan string
	Error  chan bool
}

func newFakeHandler() *fakeHandler {
	c := make(chan string, 1)
	c <- ""
	return &fakeHandler{c, make(chan bool, 1)}
}

func (h *fakeHandler) SetUp(outChannel chan<- *Message) chan bool {
	h.status <- "setup"
	return h.Error
}

func (h *fakeHandler) TearDown() {
	h.status <- "teardown"
}

func (h *fakeHandler) Handle(message *Message) {
	h.status <- "handle"
}

func (h *fakeHandler) getStatus() string {
	select {
	case s := <-h.status:
		return s

	case <-time.After(200 * time.Millisecond):
		return "blocked"
	}
}

func testListener(t *testing.T, f func(h *fakeHandler, r *Remote, l *Listener)) {
	h := newFakeHandler()
	_, client := net.Pipe()
	r := NewRemote(client, 2*time.Second)

	l := NewListener([]Handler{h})
	if h.getStatus() != "" {
		t.Error("handler has not been initialized correctly")
	}

	go l.Listen(r)
	if s := h.getStatus(); s != "setup" {
		t.Error("Handler status is not setup: ", s)
	}

	f(h, r, l)
}

func TestListener(t *testing.T) {
	testListener(t, func(h *fakeHandler, r *Remote, l *Listener) {
		r.InBuffer <- NewMessage(MESSAGE, uuid.Nil, uuid.Nil, []byte{})
		if s := h.getStatus(); s != "handle" {
			t.Error("Handler is not handling incoming message: ", s)
		}

		r.conn.Close()
		if s := h.getStatus(); s != "teardown" {
			t.Error("Handler status is not teardown: ", s)
		}
	})
}

func TestListenerError(t *testing.T) {
	testListener(t, func(h *fakeHandler, r *Remote, l *Listener) {
		h.Error <- true
		if s := h.getStatus(); s != "teardown" {
			t.Error("Error in handler should stop the connection: ", s)
		}
	})
}
