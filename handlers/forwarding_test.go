package handlers

import (
	"iosomething"
	"testing"

	"time"

	uuid "github.com/satori/go.uuid"
)

type Peer struct {
	Identity uuid.UUID
	Channel  chan *iosomething.Message
}

type TestData struct {
	Description string
	Peers       map[uuid.UUID]chan *iosomething.Message
	Sender      Peer
	ReceiverId  uuid.UUID
}

var uuidPool []uuid.UUID

func init() {
	uuidPool = make([]uuid.UUID, 10)
	for i := 0; i < 10; i++ {
		uuidPool[i] = uuid.NewV4()
	}
}

func createForwardingHandler(p map[uuid.UUID]chan *iosomething.Message) iosomething.Handler {
	ct := map[uuid.UUID]chan<- *iosomething.Message{}
	for k, v := range p {
		ct[k] = v
	}
	return NewForwardingHandler(ct)
}

func hasData(c chan *iosomething.Message) bool {
	select {
	case <-c:
		return true

	case <-time.After(200 * time.Millisecond):
		return false
	}
}

func TestConnectDisconnect(t *testing.T) {
	for _, d := range []struct {
		Description string
		Peers       map[uuid.UUID]chan<- *iosomething.Message
		Sender      uuid.UUID
	}{
		{
			"First entry",
			map[uuid.UUID]chan<- *iosomething.Message{},
			uuidPool[0],
		},
		{
			"More entries",
			map[uuid.UUID]chan<- *iosomething.Message{
				uuidPool[0]: make(chan *iosomething.Message, 1),
				uuidPool[1]: make(chan *iosomething.Message, 1),
				uuidPool[2]: make(chan *iosomething.Message, 1),
			},
			uuidPool[3],
		},
	} {
		h := NewForwardingHandler(d.Peers)
		fh := h.(*forwarding)
		h.SetUp(make(chan *iosomething.Message, 1))
		initialLength := len(fh.peers)

		h.Handle(iosomething.NewMessage(iosomething.MESSAGE, d.Sender, uuid.Nil, []byte{}))
		if len(fh.peers) != initialLength+1 {
			t.Errorf("%v error: Peer not added", d.Description)
		}

		h.TearDown()
		if len(fh.peers) != initialLength {
			t.Errorf("%v error: Peer not removed", d.Description)
		}
	}
}

// TestNotForwarding is testing cases where message should not be forwarded to anyone
func TestNotForwarding(t *testing.T) {
	for _, d := range []TestData{
		{
			"Null receiver",
			map[uuid.UUID]chan *iosomething.Message{uuid.Nil: make(chan *iosomething.Message, 1)},
			Peer{uuid.NewV4(), make(chan *iosomething.Message, 1)},
			uuid.Nil,
		},
		{
			"Null sender",
			map[uuid.UUID]chan *iosomething.Message{uuidPool[0]: make(chan *iosomething.Message, 1)},
			Peer{uuid.Nil, make(chan *iosomething.Message, 1)},
			uuidPool[0],
		},
		{
			"Empty peer list",
			map[uuid.UUID]chan *iosomething.Message{},
			Peer{uuidPool[0], make(chan *iosomething.Message, 1)},
			uuidPool[1],
		},
		{
			"Receiver not in peer list",
			map[uuid.UUID]chan *iosomething.Message{
				uuidPool[0]: make(chan *iosomething.Message, 1),
				uuidPool[1]: make(chan *iosomething.Message, 1),
			},
			Peer{uuidPool[2], make(chan *iosomething.Message, 1)},
			uuidPool[3],
		},
	} {
		h := createForwardingHandler(d.Peers)
		h.SetUp(d.Sender.Channel)
		defer h.TearDown()

		h.Handle(iosomething.NewMessage(iosomething.MESSAGE, d.Sender.Identity, d.ReceiverId, []byte{}))
		if hasData(d.Peers[d.ReceiverId]) {
			t.Errorf("%s error", d.Description)
		}
	}
}

func TestForwarding(t *testing.T) {
	for _, d := range []TestData{
		{
			"Single peer",
			map[uuid.UUID]chan *iosomething.Message{uuidPool[0]: make(chan *iosomething.Message, 1)},
			Peer{uuidPool[1], make(chan *iosomething.Message, 1)},
			uuidPool[0],
		},
		{
			"More peers",
			map[uuid.UUID]chan *iosomething.Message{
				uuidPool[0]: make(chan *iosomething.Message, 1),
				uuidPool[1]: make(chan *iosomething.Message, 1),
				uuidPool[2]: make(chan *iosomething.Message, 1),
			},
			Peer{uuidPool[3], make(chan *iosomething.Message, 1)},
			uuidPool[1],
		},
	} {
		h := createForwardingHandler(d.Peers)
		h.SetUp(d.Sender.Channel)
		defer h.TearDown()

		sent := iosomething.NewMessage(iosomething.MESSAGE, d.Sender.Identity, d.ReceiverId, []byte{})
		h.Handle(sent)
		select {
		case received := <-d.Peers[d.ReceiverId]:
			if !iosomething.Equal(sent, received) {
				t.Errorf("Message mismatch\nexpecting: %v\ngot:       %v", sent, received)
			}

		case <-time.After(200 * time.Millisecond):
			t.Errorf("%s error: Message not forwarded", d.Description)
		}
	}
}
