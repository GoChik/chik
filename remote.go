package iosomething

import (
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/cskr/pubsub"
)

// BufferSize is the size of channel buffers
const BufferSize = 10

// WriteTimeout defines the time after which a write operation is considered failed
const WriteTimeout = 1 * time.Minute

// Remote represents a remote endpoint, data can be sent or received through
// InBuffer and OutBuffer
type Remote struct {
	conn   net.Conn
	PubSub *pubsub.PubSub
}

// NewRemote creates a new Remote
func NewRemote(conn net.Conn, readTimeout time.Duration) *Remote {
	remote := Remote{
		conn:   conn,
		PubSub: pubsub.New(BufferSize),
	}

	// Send function
	go func() {
		logrus.Debug("Sender started")
		// heartbeat outgoing messages have a special type in order to avoid being bounced back
		out := remote.PubSub.Sub("out")
		for data := range out {
			message := data.(*Message)
			logrus.Debug("Sending message", message)
			conn.SetWriteDeadline(time.Now().Add(WriteTimeout))
			_, err := remote.conn.Write(message.Bytes())
			if err != nil {
				logrus.Warn("Cannot write data, exiting:", err)
				remote.Terminate()
				return
			}
		}
	}()

	// Receive function
	go func() {
		logrus.Debug("Receivr started")
		for {
			if readTimeout != 0 {
				conn.SetReadDeadline(time.Now().Add(readTimeout))
			}

			message, err := ParseMessage(conn)
			if err != nil {
				logrus.Error("Invalid message:", err)
				remote.Terminate()
				return
			}
			id, _ := message.SenderUUID()
			logrus.Debug("Message received from:", id)
			remote.PubSub.Pub(message, "in", message.Type().String())
		}
	}()

	return &remote
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	r.conn.Close()
	r.PubSub.Shutdown()
}
