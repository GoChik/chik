package utils

import (
	"encoding/binary"
	"errors"
	"net"

	"github.com/Sirupsen/logrus"
)

type Remote struct {
	Conn   net.Conn
	buffer chan []byte
}

func max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

// NewRemote creates a new remote
func NewRemote(conn net.Conn) Remote {
	remote := Remote{}
	remote.Conn = conn
	remote.buffer = make(chan []byte, 10)

	go func() {
		for {
			data, more := <-remote.buffer

			if !more {
				logrus.Debug("Channel closed, exiting")
				return
			}

			err := binary.Write(remote.Conn, binary.BigEndian, data)
			if err != nil {
				logrus.Warn("Cannot write data, exiting:", err)
				return
			}
		}
	}()

	return remote
}

// SendMessage enqueue message to be sent by the remote
func (r *Remote) SendMessage(data []byte) error {
	select {
	case r.buffer <- data:
		return nil
	default:
		return errors.New("Write buffer is full")
	}
}

// Terminate closes the connection and the send channel
func (r *Remote) Terminate() {
	r.Conn.Close()
}
