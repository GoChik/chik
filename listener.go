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

	// error channel
	errorChannel := make(chan bool, 1)

	// starting handlers
	for _, h := range l.handlers {
		ec := h.SetUp(remote.OutBuffer)
		defer h.TearDown()

		go func(c chan bool) {
			errorChannel <- <-c
		}(ec)
	}

	for {
		select {
		case <-stop:
			logrus.Debug("stopping remote routine")
			return

		case <-errorChannel:
			logrus.Error("Handler error detected")
			remote.Terminate()
			break

		case message := <-remote.InBuffer:
			// handling message
			for _, h := range l.handlers {
				h.Handle(message)
			}
			break
		}
	}
}
