// +build platform_unipi_neuron

package platform

import (
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/bus/softbus"
	"github.com/gochik/chik/handlers/io/bus/unipibus"
	"github.com/gochik/chik/handlers/io/bus/w1bus"
)

type unipiPlatform struct{}

func init() {
	for _, bus := range []func() bus.Bus{
		softbus.New,
		unipibus.New,
		w1bus.New,
	} {
		buses = append(buses, bus)
	}
}
