// +build platform_soft

package platform

import (
	"github.com/gochik/chik/handlers/io/bus/softbus"
)

func init() {
	initializeBus(softbus.New)
}
