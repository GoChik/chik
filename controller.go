package chik

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/cskr/pubsub"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// BufferSize is the size of channel buffers
const BufferSize = 10

// MaxIdleTime is the maximum time to wait before closing a connection for inactivity
const MaxIdleTime = 5 * time.Minute

// LoopbackID is the id internal only messages are sent to
var LoopbackID = uuid.Nil

type Controller struct {
	ID         uuid.UUID
	pubSub     *pubsub.PubSub
	remote     *Remote
	disconnect sync.Mutex
}

// NewController creates a new controller
func NewController() *Controller {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var levelString string
	config.GetStruct("log_level", &levelString)
	logLevel, err := zerolog.ParseLevel(levelString)
	if err != nil {
		log.Warn().Msgf("Cannot parse log level, setting it to warning by default: %s", err)
		config.Set("log_level", "debug")
		config.Sync()
		logLevel = zerolog.WarnLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	var idString string
	config.GetStruct("identity", &idString)
	identity := uuid.FromStringOrNil(idString)
	if identity == uuid.Nil {
		identity, _ = uuid.NewV1()
		config.Set("identity", identity)
		config.Sync()
		log.Warn().Msg("Cannot get identity from config file, one has been auto generated")
	}
	log.Info().Str("identity", identity.String())

	return &Controller{
		ID:     identity,
		pubSub: pubsub.New(BufferSize),
	}
}

func topicsAsStrings(topics []types.CommandType) []string {
	result := make([]string, len(topics))
	for _, topic := range topics {
		result = append(result, topic.String())
	}
	return result
}

func (c *Controller) run(ctx context.Context, g *errgroup.Group, h Handler) {
	log.Info().Str("handler", h.String()).Msgf("Starting %s handler", h.String())
	timer, err := h.Setup(c)
	if err != nil {
		log.Err(err).Str("handler", h.String())
		return
	}
	subscribedTopics := c.Sub(topicsAsStrings(h.Topics())...)
	g.Go(func() error {
		stop := func() {
			timer.ticker.Stop()
			h.Teardown()
			log.Debug().
				Str("handler", h.String()).
				Msg("Stopping handler")
		}
		if timer.triggerAtStart {
			h.HandleTimerEvent(time.Now(), c)
		}
		for {
			select {
			case <-ctx.Done():
				stop()
				return ctx.Err()
			case rawMessage, ok := <-subscribedTopics:
				if !ok {
					log.Error().
						Str("handler", h.String()).
						Msg("Message channel closed. Quitting")
					stop()
					return fmt.Errorf("Channel closed on handler %v", h.String())
				}
				err := h.HandleMessage(rawMessage.(*Message), c)
				if err != nil {
					return err
				}

			case tick := <-timer.ticker.C:
				h.HandleTimerEvent(tick, c)
			}
		}
	})
}

// Start starts every registered handler
func (c *Controller) Start(ctx context.Context, handlers []Handler) {
	// TODO: order handlers by dependencies
	g, ctx := errgroup.WithContext(ctx)
	for _, h := range handlers {
		c.run(ctx, g, h)
	}
	g.Wait()
	log.Info().Msg("Controller terminated")
	c.Disconnect()
	c.pubSub.Shutdown()
}

// Connect tries to brign up the remoe connection
// it returns a channel that gets closed when the connection goes down
func (c *Controller) Connect(connection net.Conn) <-chan bool {
	c.Disconnect()
	c.remote = newRemote(c, connection, MaxIdleTime)
	return c.remote.Closed
}

// Disconnect disconnects the remote connection (if any)
func (c *Controller) Disconnect() {
	c.disconnect.Lock()
	if c.remote != nil {
		c.remote.Terminate()
		c.remote = nil
	}
	c.disconnect.Unlock()
}

// PubMessage publishes a Message
func (c *Controller) PubMessage(message *Message, topics ...string) {
	c.pubSub.TryPub(message, topics...)
}

// Pub publishes a Message composed by the given Command
func (c *Controller) Pub(command *types.Command, receiverID uuid.UUID) {
	messageKind := types.AnyOutgoingCommandType.String()
	if receiverID == LoopbackID {
		messageKind = command.Type.String()
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
	command := types.NewCommand(replyType, replyContent)

	// If sender is null the message is internal, otherwise it needs to go out
	c.Pub(command, receiver)
}
