//go:generate dbus-codegen-go -package=systemd -camelize -client-only -only=org.freedesktop.systemd1.Manager -only=org.freedesktop.DBus.Properties -only=org.freedesktop.systemd1.Unit -output dbusInterface.go xml_interfaces/org.freedesktop.systemd1.unit.snapserver_2eservice.xml xml_interfaces/org.freedesktop.systemd1.xml

package systemd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/godbus/dbus/v5"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "systemd").Logger()

const (
	SystemdObjectPath    = "/org/freedesktop/systemd1"
	SystemdInterfaceName = "org.freedesktop.systemd1"
)

type SystemdRequestCommand struct {
	Action      types.Action `json:"action"`
	ServiceName string       `json:"service_name"`
}

type SystemdReplyCommand struct {
	ServiceName string `json:"service_name"`
	Status      string `json:"status"`
}

// Systemd handler
type Systemd struct {
	chik.BaseHandler
	connection *dbus.Conn
}

// New creates a telegram handler. useful for sending notifications about events
func New() *Systemd {
	return &Systemd{}
}

func (h *Systemd) Dependencies() []string {
	return []string{}
}

func (h *Systemd) Topics() []types.CommandType {
	return []types.CommandType{types.SystemdRequestCommandType}
}

func (h *Systemd) Setup(controller *chik.Controller) (t chik.Interrupts, err error) {
	h.connection, err = dbus.SystemBus()
	return chik.Interrupts{Timer: chik.NewEmptyTimer()}, err
}

type SignalListener struct {
	C    <-chan *dbus.Signal
	Stop func()
}

func (h *Systemd) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	var command SystemdRequestCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logger.Warn().Msg("Unexpected message")
		return nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
	defer cancel()

	manager := NewOrgFreedesktopSystemd1Manager(h.connection.Object(SystemdInterfaceName, SystemdObjectPath))
	servicePath, err := manager.GetUnit(ctx, command.ServiceName+".service")
	if err != nil {
		logger.Err(err).Msg("failed getting specified service")
		return nil
	}
	unit := NewOrgFreedesktopSystemd1Unit(h.connection.Object(SystemdInterfaceName, servicePath))
	activeState, err := unit.GetActiveState(ctx)
	if err != nil {
		logger.Err(err).Msg("failed to get active state")
		activeState = "unknown"
	}

	switch command.Action {
	case types.SET:
		if activeState != "active" {
			unit.Start(ctx, "replace")
			activeState = "active"
		}

	case types.RESET:
		if activeState != "inactive" && activeState != "failed" {
			unit.Stop(ctx, "replace")
			activeState = "inactive"
		}
	}

	controller.Reply(message, types.SystemdReplyCommandType, SystemdReplyCommand{
		ServiceName: command.ServiceName,
		Status:      activeState,
	})

	return nil
}

func (h *Systemd) Teardown() {
	h.connection.Close()
}

func (h *Systemd) String() string {
	return "systemd"
}
