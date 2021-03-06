package chik

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gochik/chik/types"
)

type Timer struct {
	triggerAtStart bool
	ticker         *time.Ticker
}

// NewTimer creates a new timer given an interval and the option to fire when started
func NewTimer(interval time.Duration, triggerAtStart bool) Timer {
	return Timer{
		triggerAtStart,
		time.NewTicker(interval),
	}
}

// NewEmptyTimer creates a timer that does never fire
func NewEmptyTimer() Timer {
	return Timer{
		false,
		&time.Ticker{C: make(chan time.Time, 0)},
	}
}

// Handler is the interface that handles network messages
// and optionally can return a reply
type Handler interface {
	fmt.Stringer
	Dependencies() []string
	Topics() []types.CommandType
	Setup(controller *Controller) Timer
	HandleMessage(message *Message, controller *Controller) error
	HandleTimerEvent(tick time.Time, controller *Controller)
	Teardown()
}

// StatusHolder is a struct that stores status of an handler that needs to trigger status changes
// when something happens
type StatusHolder struct {
	status     interface{}
	moduleName string
}

// NewStatusHolder creates a StatusHolder
func NewStatusHolder(moduleName string) *StatusHolder {
	return &StatusHolder{
		moduleName: moduleName,
	}
}

func (s *StatusHolder) emitStatusChanged(c *Controller) {
	status := types.Status{}
	status[s.moduleName] = s.status
	c.Pub(types.NewCommand(types.StatusUpdateCommandType, status), LoopbackID)
}

// Set stores the status and emits it if it is changed
func (s *StatusHolder) Set(status interface{}, controller *Controller) {
	if reflect.DeepEqual(s.status, status) {
		return
	}
	s.status = status
	s.emitStatusChanged(controller)
}

// Edit the current status via editFunction
func (s *StatusHolder) Edit(controller *Controller, editFunction func(interface{}) interface{}) {
	newStatus := editFunction(s.status)
	if reflect.DeepEqual(newStatus, s.status) {
		return
	}
	s.status = newStatus
	s.emitStatusChanged(controller)
}
