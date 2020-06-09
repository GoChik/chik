package chik

import (
	"net"
	"sync"
	"time"

	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"
)

var remotelog = log.With().Str("component", "remote").Logger()

// WriteTimeout defines the time after which a write operation is considered failed
const WriteTimeout = 1 * time.Minute

// Remote represents a remote endpoint, data can be sent or received through
// InBuffer and OutBuffer
type Remote struct {
	Closed   chan bool
	conn     net.Conn
	stopOnce sync.Once
}

// NewRemote creates a new Remote
func newRemote(controller *Controller, conn net.Conn, readTimeout time.Duration) *Remote {
	remote := Remote{
		Closed: make(chan bool, 1),
		conn:   conn,
	}

	// Send function
	go func() {
		remotelog.Info().Msg("Sender started")
		out := controller.Sub(types.AnyOutgoingCommandType.String())
		for data := range out {
			message, ok := data.(*Message)
			if !ok {
				remotelog.Warn().Msg("Trying to something that's not a message")
				continue
			}
			if message.sender == uuid.Nil {
				message.sender = controller.ID
			}
			remotelog.Debug().Msgf("Sending message: %v", message)
			conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
			data, err := message.Bytes()
			if err != nil {
				remotelog.Warn().Msgf("Cannot encode message: %v", err)
			}
			_, err = remote.conn.Write(data)
			if err != nil {
				log.Warn().Msgf("Cannot write data, exiting: %v", err)
				remote.Terminate()
				break
			}
		}
		remotelog.Info().Msg("Sender terminated")
	}()

	// Receive function
	go func() {
		remotelog.Info().Msg("Receiver started")
		for {
			if readTimeout != 0 {
				conn.SetReadDeadline(time.Now().Add(readTimeout))
			}

			message, err := ParseMessage(conn)
			if err != nil {
				remotelog.Error().Msgf("Invalid message: %v", err)
				remote.Terminate()
				break
			}
			remotelog.Debug().Msgf("Message received: %v", message)
			controller.PubMessage(message, types.AnyIncomingCommandType.String(), message.Command().Type.String())
		}
		remotelog.Info().Msg("Receiver terminated")
	}()

	return &remote
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	r.stopOnce.Do(func() {
		r.conn.Close()
		r.Closed <- true
		close(r.Closed)
	})
}
