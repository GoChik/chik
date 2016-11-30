package utils

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
)

type Remote struct {
	Conn  net.Conn
	mutex sync.Mutex
}

func max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}

func NewRemote(conn net.Conn) Remote {
	remote := Remote{}
	remote.mutex = sync.Mutex{}
	remote.Conn = conn

	return remote
}

func (r *Remote) SendMessage(kind MsgType, content ...interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	err := binary.Write(r.Conn, binary.BigEndian, kind)
	if err != nil {
		return err
	}

	size := 0
	for _, data := range content {
		size += max(binary.Size(data), 0)
	}
	log.Println("Writing message of size: ", size)
	err = binary.Write(r.Conn, binary.BigEndian, uint32(size))
	if err != nil {
		return err
	}

	if size > 0 {
		for _, data := range content {
			err = binary.Write(r.Conn, binary.BigEndian, data)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
