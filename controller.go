package chik

import (
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/cskr/pubsub"
	"github.com/gofrs/uuid"
)

// BufferSize is the size of channel buffers
const BufferSize = 10

const MaxIdleTime = 5 * time.Minute

type Controller struct {
	ID       uuid.UUID
	PubSub   *pubsub.PubSub
	remote   *Remote
	shutdown sync.Once
	handlers sync.WaitGroup
	active   bool
}

func NewController(id uuid.UUID) *Controller {
	return &Controller{
		active: true,
		ID:     id,
		PubSub: pubsub.New(BufferSize),
	}
}

func (c *Controller) Start(h Handler) {
	c.handlers.Add(1)
	go func() {
		for c.active {
			h.Run(c)
		}
		c.handlers.Done()
	}()
}

func (c *Controller) Connect(connection net.Conn) <-chan bool {
	if c.remote != nil {
		c.remote.Terminate()
	}
	c.remote = NewRemote(c, connection, MaxIdleTime)
	return c.remote.Closed
}

func (c *Controller) Disconnect() {
	c.remote.Terminate()
}

// Reply sends back a reply message
func (c *Controller) Reply(request *Message, replyType MsgType, replyContent interface{}) {
	rawReply, err := json.Marshal(replyContent)
	if err != nil {
		logrus.Error("Cannot marshal status message")
		return
	}

	sender := request.SenderUUID()
	reply := NewMessage(replyType, sender, rawReply)

	// If sender is null the message is internal, otherwise it needs to go out
	destination := "out"
	if sender == uuid.Nil {
		destination = replyType.String()
	}
	c.PubSub.Pub(reply, destination)
}

func (c *Controller) Shutdown() {
	c.Disconnect()
	c.shutdown.Do(func() {
		c.active = false
		c.PubSub.Shutdown()
		c.handlers.Wait()
	})
}
