package chik

import (
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"
)

// Handler is the interface that handles network messages
// and optionally can return a reply
type Handler interface {
	fmt.Stringer
	Run(controller *Controller)
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
	status := Status{}
	status[s.moduleName] = s.status
	c.PubSub.Pub(NewMessage(uuid.Nil, NewCommand(StatusUpdateCommandType, status)), StatusUpdateCommandType.String())
}

// SetStatus stores the status and emits it if it is changed
func (s *StatusHolder) SetStatus(status interface{}, controller *Controller) {
	if reflect.DeepEqual(s.status, status) {
		return
	}
	s.status = status
	s.emitStatusChanged(controller)
}
