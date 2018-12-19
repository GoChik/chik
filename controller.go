package chik

import (
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/cskr/pubsub"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
)

// BufferSize is the size of channel buffers
const BufferSize = 10
const MaxIdleTime = 5 * time.Minute

// LoopbackID is the id internal only messages are sent to
var LoopbackID = uuid.Nil

type Controller struct {
	ID       uuid.UUID
	pubSub   *pubsub.PubSub
	remote   *Remote
	shutdown sync.Once
	handlers sync.WaitGroup
	active   bool
}

// NewController creates a new controller
func NewController() *Controller {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})

	logLevel, err := logrus.ParseLevel(config.Get("log_level").(string))
	if err != nil {
		logrus.Warn("Cannot parse log level, setting it to warning by default: ", err)
		config.Set("log_level", "warning")
		config.Sync()
		logLevel = logrus.WarnLevel
	}
	logrus.SetLevel(logLevel)

	identity := uuid.FromStringOrNil(config.Get("identity").(string))
	if identity == uuid.Nil {
		id, _ := uuid.NewV1()
		config.Set("identity", id)
		config.Sync()
		logrus.Warn("Cannot get identity from config file, one has been auto generated")
	}

	return &Controller{
		active: true,
		ID:     identity,
		pubSub: pubsub.New(BufferSize),
	}
}

// Start starts every registered handler
func (c *Controller) Start(h Handler) {
	c.handlers.Add(1)
	go func() {
		for c.active {
			h.Run(c)
		}
		c.handlers.Done()
	}()
}

// Connect tries to brign up the remoe connection
// it returns a channel that gets closed when the connection goes down
func (c *Controller) Connect(connection net.Conn) <-chan bool {
	if c.remote != nil {
		c.remote.Terminate()
	}
	c.remote = newRemote(c, connection, MaxIdleTime)
	return c.remote.Closed
}

// Disconnect disconnects the remote connection (if any)
func (c *Controller) Disconnect() {
	c.remote.Terminate()
}

// PubMessage publishes a Message
func (c *Controller) PubMessage(message *Message, topics ...string) {
	c.pubSub.TryPub(message, topics...)
}

// Pub publishes a Message composed by the given Command
func (c *Controller) Pub(command *types.Command, receiverID uuid.UUID) {
	messageKind := command.Type.String()
	if receiverID == LoopbackID {
		messageKind = OutgoingMessage
	}

	c.PubMessage(NewMessage(receiverID, command), messageKind)
}

// Sub Subscribes to one or more message types
func (c *Controller) Sub(topics ...string) chan interface{} {
	return c.pubSub.Sub(topics...)
}

// SubOnce subscribes to the first event of one of the given topics, then it deletes the subscription
func (c *Controller) SubOnce(topics ...string) chan interface{} {
	return c.pubSub.SubOnce(topics...)
}

// Reply sends back a reply message
func (c *Controller) Reply(request *Message, replyType types.CommandType, replyContent interface{}) {
	receiver := request.SenderUUID()
	reply := NewMessage(receiver, types.NewCommand(replyType, replyContent))

	// If sender is null the message is internal, otherwise it needs to go out
	destination := OutgoingMessage
	if receiver == uuid.Nil {
		destination = replyType.String()
	}
	c.PubMessage(reply, destination)
}

// Shutdown disconnects and turns off every handler
func (c *Controller) Shutdown() {
	c.Disconnect()
	c.shutdown.Do(func() {
		c.active = false
		c.pubSub.Shutdown()
		c.handlers.Wait()
	})
}
