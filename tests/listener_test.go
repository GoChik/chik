package tests

import (
	"iosomething"
	"net"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
)

type fakeHandler struct {
	status chan string
}

func newFakeHandler() *fakeHandler {
	c := make(chan string, 1)
	c <- ""
	return &fakeHandler{c}
}

func (h *fakeHandler) SetUp(outChannel chan<- *iosomething.Message) chan bool {
	h.status <- "setup"
	return make(chan bool, 0)
}

func (h *fakeHandler) TearDown() {
	h.status <- "teardown"
}

func (h *fakeHandler) Handle(message *iosomething.Message) {
	h.status <- "handle"
}

func (h *fakeHandler) getStatus() string {
	select {
	case s := <-h.status:
		return s

	case <-time.After(500 * time.Millisecond):
		return "blocked"
	}
}

func TestListener(t *testing.T) {
	h := newFakeHandler()
	server, client := net.Pipe()
	r := iosomething.NewRemote(client, 2*time.Second)

	l := iosomething.NewListener([]iosomething.Handler{h})
	if h.getStatus() != "" {
		t.Error("handler has not been initialized correctly")
	}

	go l.Listen(r)
	if s := h.getStatus(); s != "setup" {
		t.Error("Handler status is not setup: ", s)
	}

	r.InBuffer <- iosomething.NewMessage(iosomething.MESSAGE, uuid.Nil, uuid.Nil, []byte{})
	if s := h.getStatus(); s != "handle" {
		t.Error("Handler is not handling incoming message: ", s)
	}

	server.Close()
	if s := h.getStatus(); s != "teardown" {
		t.Error("Handler status is not teardown: ", s)
	}
}
