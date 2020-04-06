// +build platform_raspberrypi

package platform

import (
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/bus/gpiobus"
	"github.com/gochik/chik/handlers/io/bus/softbus"
)

type unipiPlatform struct{}

func init() {
	for _, bus := range []func() bus.Bus{
		softbus.New,
		gpiobus.New,
	} {
		buses = append(buses, bus)
	}
}
