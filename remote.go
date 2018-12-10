package chik

import (
	"net"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

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
		logrus.Debug("Sender started")
		// heartbeat outgoing messages have a special type in order to avoid being bounced back
		out := controller.PubSub.Sub("out")
		for data := range out {
			message, ok := data.(*Message)
			if !ok {
				logrus.Warn("Trying to something that's not a message")
				continue
			}
			if message.sender == uuid.Nil {
				message.sender = controller.ID
			}
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
			id := message.SenderUUID()
			logrus.Debug("Message received from:", id)
			controller.PubSub.TryPub(message, "in", message.Command().Type.String())
		}
	}()

	return &remote
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	r.stopOnce.Do(func() {
		r.conn.Close()
		r.Closed <- true
	})
}
