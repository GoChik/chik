package iosomething

import (
	"github.com/Sirupsen/logrus"
)

type Listener struct {
	handlers []Handler
}

func NewListener(handlers []Handler) *Listener {
	return &Listener{handlers}
}

func (l *Listener) Listen(remote *Remote) {
	stop := remote.StopChannel()
	defer close(remote.OutBuffer)

	// starting handlers
	for _, h := range l.handlers {
		h.SetUp(remote.OutBuffer)
		defer h.TearDown()
	}

	for {
		select {
		case <-stop:
			logrus.Debug("stopping remote routine")
			return

		case message := <-remote.InBuffer:
			// handling message
			for _, h := range l.handlers {
				h.Handle(message)
			}
			break
		}
	}
}