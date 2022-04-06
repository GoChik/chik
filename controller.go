package chik

import (
	"context"
	"sync"
	"time"

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
const MaxIdleTime = 2 * time.Minute

// LoopbackID is the id internal only messages are sent to
var LoopbackID = uuid.Nil

type Timer struct {
	triggerAtStart bool
	ticker         *time.Ticker
}

// NewTimer creates a new timer given an interval and the option to fire when started
func NewTimer(interval time.Duration, triggerAtStart bool) Timer {
	return Timer{
		triggerAtStart,
		time.NewTicker(interval),
	}
}

// NewStartupActionTimer creates a timer that fires only at start and then never triggers again
func NewStartupActionTimer() Timer {
	return Timer{
		true,
		&time.Ticker{C: make(chan time.Time, 0)},
	}
}

// NewEmptyTimer creates a timer that does never fire
func NewEmptyTimer() Timer {
	return Timer{
		false,
		&time.Ticker{C: make(chan time.Time, 0)},
	}
}

type Interrupts struct {
	Timer Timer
	Event <-chan interface{}
}

type Controller struct {
	ID     uuid.UUID
	pubSub *pubsub.PubSub
	wg     sync.WaitGroup
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

func (c *Controller) runHandler(ctx context.Context, h Handler) (subContext context.Context) {
	subContext, cancel := context.WithCancel(ctx)
	interrupts, err := h.Setup(c)
	if err != nil {
		log.Err(err).Str("handler", h.String()).Msg("setup error")
		cancel()
		return
	}
	subscribedTopics := c.Sub(topicsAsStrings(h.Topics())...)
	go func() {
		defer func() {
			c.Unsub(subscribedTopics)
			interrupts.Timer.ticker.Stop()
			h.Teardown()
			cancel()
		}()

		if interrupts.Timer.triggerAtStart {
			if err := h.HandleTimerEvent(time.Now(), c); err != nil {
				log.Err(err).Str("handler", h.String()).Msg("Error during first timer call")
				return
			}
		}
		for {
			select {
			case <-ctx.Done():
				return

			case rawMessage, ok := <-subscribedTopics:
				if !ok {
					log.Error().
						Str("handler", h.String()).
						Msg("Message channel closed. Quitting")
					return
				}

				if err := h.HandleMessage(rawMessage.(*Message), c); err != nil {
					log.Err(err).Str("handler", h.String()).Msg("Error during first timer call")
					return
				}

			case tick := <-interrupts.Timer.ticker.C:
				if err := h.HandleTimerEvent(tick, c); err != nil {
					return
				}

			case event := <-interrupts.Event:
				if err := h.HandleChannelEvent(event, c); err != nil {
					return
				}
			}
		}
	}()

	return
}

func (c *Controller) executeHandler(ctx context.Context, h Handler) (err error) {
	log.Info().Str("handler", h.String()).Msg("Starting handler")
	defer func() {
		log.Info().Str("handler", h.String()).Msg("Stopping handler")
		c.wg.Done()
	}()

	subctx := c.runHandler(ctx, h)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-subctx.Done():
			<-time.After(5 * time.Second)
			c.wg.Add(1)
			go c.executeHandler(ctx, h)
			return
		}
	}
}

// Start starts every registered handler
func (c *Controller) Start(ctx context.Context, handlers []Handler) {
	// TODO: order handlers by dependencies
	for _, h := range handlers {
		c.wg.Add(1)
		go c.executeHandler(ctx, h)
	}
	c.wg.Wait()
	log.Info().Msg("Controller terminated")
	c.pubSub.Shutdown()
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

func (c *Controller) Unsub(subscription chan interface{}) {
	for {
		select {
		case <-subscription:
			continue

		default:
			c.pubSub.Unsub(subscription)
			return
		}
	}
}

// Reply sends back a reply message
func (c *Controller) Reply(request *Message, replyType types.CommandType, replyContent interface{}) {
	receiver := request.SenderUUID()
	command := types.NewCommand(replyType, replyContent)

	// If sender is null the message is internal, otherwise it needs to go out
	c.Pub(command, receiver)
}
