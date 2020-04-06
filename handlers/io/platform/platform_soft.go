// +build platform_soft

package platform

import (
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/bus/softbus"
)

type unipiPlatform struct{}

func init() {
	for _, bus := range []func() bus.Bus{
		softbus.New,
	} {
		buses = append(buses, bus)
	}
}
