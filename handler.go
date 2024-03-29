package chik

import (
	"fmt"
	"reflect"
	"time"

	"github.com/gochik/chik/types"
)

// Handler is the interface that handles network messages
// and optionally can return a reply
type Handler interface {
	fmt.Stringer
	Dependencies() []string
	Topics() []types.CommandType
	Setup(controller *Controller) (Interrupts, error)
	HandleMessage(message *Message, controller *Controller) error
	HandleTimerEvent(tick time.Time, controller *Controller) error
	HandleChannelEvent(event interface{}, controller *Controller) error
	Teardown()
}

type BaseHandler struct{}

func (s *BaseHandler) String() string {
	return ""
}

func (s *BaseHandler) Dependencies() []string {
	return []string{}
}

func (s *BaseHandler) Topics() []types.CommandType {
	return []types.CommandType{}
}

func (s *BaseHandler) Setup(controller *Controller) (Interrupts, error) {
	return Interrupts{Timer: NewEmptyTimer()}, nil
}

func (s *BaseHandler) HandleMessage(message *Message, controller *Controller) error {
	return nil
}

func (s *BaseHandler) HandleTimerEvent(tick time.Time, controller *Controller) error {
	return nil
}

func (s *BaseHandler) HandleChannelEvent(event interface{}, controller *Controller) error {
	return nil
}

func (s *BaseHandler) Teardown() {}

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

func (s *StatusHolder) Get() interface{} {
	return s.status
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
func (s *StatusHolder) Edit(controller *Controller, editFunction func(interface{}) (interface{}, bool)) {
	newStatus, forceChange := editFunction(s.status)

	if !forceChange && reflect.DeepEqual(newStatus, s.status) {
		return
	}
	s.status = newStatus
	s.emitStatusChanged(controller)
}
