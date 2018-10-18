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

// WriteTimeout defines the time after which a write operation is considered failed
const WriteTimeout = 1 * time.Minute

// Remote represents a remote endpoint, data can be sent or received through
// InBuffer and OutBuffer
type Remote struct {
	PubSub   *pubsub.PubSub
	id       uuid.UUID
	conn     net.Conn
	stopOnce sync.Once
}

// NewRemote creates a new Remote
func NewRemote(id uuid.UUID, conn net.Conn, readTimeout time.Duration) *Remote {
	remote := Remote{
		id:     id,
		conn:   conn,
		PubSub: pubsub.New(BufferSize),
	}

	// Send function
	go func() {
		logrus.Debug("Sender started")
		// heartbeat outgoing messages have a special type in order to avoid being bounced back
		out := remote.PubSub.Sub("out")
		for data := range out {
			message, ok := data.(*Message)
			if !ok {
				logrus.Warn("Trying to something that's not a message")
				continue
			}
			if message.sender == uuid.Nil {
				message.sender = remote.id
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
			remote.PubSub.Pub(message, "in", message.Type().String())
		}
	}()

	return &remote
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	r.stopOnce.Do(func() {
		r.conn.Close()
		r.PubSub.Shutdown()
	})
}

// Reply sends back a reply message
func (r *Remote) Reply(request *Message, replyType MsgType, replyContent interface{}) {
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
	r.PubSub.Pub(reply, destination)
}
