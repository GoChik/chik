package chik

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testHandler struct {
	BaseHandler
	trigger      chan interface{}
	currentValue string
}

func (t *testHandler) Setup(controller *Controller) (Interrupts, error) {
	return Interrupts{Timer: NewEmptyTimer(), Event: t.trigger}, nil
}

func (t *testHandler) HandleChannelEvent(event interface{}, controller *Controller) error {
	val, ok := event.(string)
	if !ok {
		return errors.New("Conversion error")
	}
	t.currentValue = val
	return nil
}

func TestHandlerError(t *testing.T) {
	cont := NewController()
	trigger := make(chan interface{}, 0)
	handler := testHandler{trigger: trigger}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cont.Start(
		ctx,
		[]Handler{&handler},
	)

	trigger <- false
	time.Sleep(6 * time.Second)
	word := "works"
	trigger <- word

	time.Sleep(10 * time.Millisecond)

	if handler.currentValue != word {
		t.Fatal("Handler error recovery failed: ", handler.currentValue)
	}
}
