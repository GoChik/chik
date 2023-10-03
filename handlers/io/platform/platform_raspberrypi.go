//go:build platform_raspberrypi
// +build platform_raspberrypi

package platform

import (
	"github.com/gochik/chik/handlers/io/bus/gpiobus"
	"github.com/gochik/chik/handlers/io/bus/softbus"
)

func init() {
	initializeBus(
		softbus.New,
		gpiobus.New,
	)
}
