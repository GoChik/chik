package utils

import (
	"bufio"
	"io"
	"net"
	"sync"

	"github.com/Sirupsen/logrus"
)

var mutex = sync.Mutex{}

const BUFFER_SIZE = 10

type Remote struct {
	conn         net.Conn
	OutBuffer    chan *Message
	InBuffer     chan *Message
	stopChannels []chan bool
}

// NewRemote creates a new Remote
func NewRemote(conn net.Conn) *Remote {
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
				message, err := ParseMessage(reader)
				if err == io.EOF {
					logrus.Debug("Connection closed")
					remote.Terminate()
					continue
				}

				if err != nil {
					logrus.Error("Invalid message:", err)
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
	mutex.Lock()
	defer mutex.Unlock()

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
