package utils

import (
	"bufio"
	"net"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

// BUFFER_SIZE is the size of remote buffers
const BUFFER_SIZE = 10
const WRITE_TIMEOUT = 1 * time.Minute

// Remote represents a remote endpoint, data can be sent or received through
// InBuffer and OutBuffer
type Remote struct {
	conn         net.Conn
	OutBuffer    chan *Message
	InBuffer     chan *Message
	stopChannels []chan bool
	mutex         sync.Mutex
}

// NewRemote creates a new Remote
func NewRemote(conn net.Conn, readTimeout time.Duration) *Remote {
	remote := Remote{
		conn:         conn,
		OutBuffer:    make(chan *Message, BUFFER_SIZE),
		InBuffer:     make(chan *Message, BUFFER_SIZE),
		stopChannels: make([]chan bool, 2),
	}

	remote.stopChannels[0] = make(chan bool, 1)
	remote.stopChannels[1] = make(chan bool, 1)

	// Send function
	go func() {
		stop := remote.stopChannels[0]
		for {
			select {
			case <-stop:
				logrus.Debug("Terminating OutBuffer")
				close(remote.OutBuffer)
				return

			case data, more := <-remote.OutBuffer:
				if !more {
					logrus.Debug("Channel closed, exiting")
					remote.Terminate()
					continue
				}

				conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT))
				_, err := remote.conn.Write(data.Bytes())
				if err != nil {
					logrus.Warn("Cannot write data, exiting:", err)
					remote.Terminate()
					continue
				}
			}
		}
	}()

	// Receive function
	go func() {
		stop := remote.stopChannels[1]
		reader := bufio.NewReader(remote.conn)
		for {
			select {
			case <-stop:
				logrus.Debug("Terminaing InBuffer")
				close(remote.InBuffer)
				return

			default:
				if readTimeout != 0 {
					conn.SetReadDeadline(time.Now().Add(readTimeout))
				}

				message, err := ParseMessage(reader)
				if err != nil {
					logrus.Error("Invalid message:", err)
					remote.Terminate()
					continue
				}

				remote.InBuffer <- message
			}
		}
	}()

	return &remote
}

// StopChannel returns a channel that gets written on Stop
func (r *Remote) StopChannel() chan bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	stop := make(chan bool, 1)
	r.stopChannels = append(r.stopChannels, stop)
	return stop
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	for _, c := range r.stopChannels {
		c <- true
		close(c)
	}

	r.conn.Close()
}
