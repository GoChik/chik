// +build platform_unipi_neuron

package platform

import (
	"github.com/gochik/chik/handlers/io/bus/softbus"
	"github.com/gochik/chik/handlers/io/bus/unipibus"
	"github.com/gochik/chik/handlers/io/bus/w1bus"
)

func init() {
	initializeBus(softbus.New, unipibus.New, w1bus.New)
}
