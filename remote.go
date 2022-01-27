package chik

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var logger = log.With().Str("component", "remote").Logger()

// Remote represents a remote endpoint, data are sent via Controller.Pub() and received directly by the interested Handler
type Remote struct {
	conn    net.Conn
	timeout time.Duration
}

func (r Remote) send(ctx context.Context, controller *Controller) error {
	logger.Info().Msg("Sender started")
	out := controller.Sub(types.AnyOutgoingCommandType.String(), types.RemoteStopCommandType.String())
	defer func() {
		controller.Unsub(out)
		logger.Info().Msg("Sender terminated")
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case data := <-out:
			message, ok := data.(*Message)
			if !ok {
				logger.Warn().Msg("Trying to something that's not a message")
				continue
			}
			if message.command.Type == types.RemoteStopCommandType {
				logger.Info().Msg("Stop command received. Terminating Sender")
				return errors.New("Stop received")
			}
			if message.sender == uuid.Nil {
				message.sender = controller.ID
			}
			logger.Debug().Msgf("Sending message: %v", message)
			r.conn.SetWriteDeadline(time.Now().Add(r.timeout))
			bytes, err := message.Bytes()
			if err != nil {
				logger.Warn().Msgf("Cannot encode message: %v", err)
			}
			_, err = r.conn.Write(bytes)
			if err != nil {
				logger.Warn().Msgf("Cannot write bytes, exiting: %v", err)
				return err
			}
		}
	}
}

func (r Remote) receive(ctx context.Context, controller *Controller) error {
	logger.Info().Msg("Receiver started")
	defer logger.Info().Msg("Receiver terminated")

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			if r.timeout != 0 {
				r.conn.SetReadDeadline(time.Now().Add(r.timeout))
			}
			message, err := ParseMessage(r.conn)
			if err != nil {
				logger.Error().Msgf("Invalid message: %v", err)
				return err
			}
			logger.Debug().Msgf("Message received: %v", message)
			controller.PubMessage(message, types.AnyIncomingCommandType.String(), message.Command().Type.String())
		}
	}
}

// Start starts a new remote and returns a context and a cancel function to stop remote operations.
// The returned context can also be closed by an error or a timeout in the send/receive routine
func StartRemote(controller *Controller, conn net.Conn, readTimeout time.Duration) (context.Context, context.CancelFunc) {
	remote := Remote{
		conn:    conn,
		timeout: readTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		g, innerCtx := errgroup.WithContext(ctx)
		// Send function
		g.Go(func() error { return remote.send(innerCtx, controller) })

		// Receive function
		g.Go(func() error { return remote.receive(innerCtx, controller) })
		g.Wait()
		cancel()
	}()

	return ctx, cancel
}
