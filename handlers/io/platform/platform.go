package platform

import (
	"github.com/gochik/chik/handlers/io/bus"
)

var buses = make([]func() bus.Bus, 0)

// CreateBuses creates the set of available actuators
func CreateBuses() map[string]bus.Bus {
	result := make(map[string]bus.Bus, len(buses))
	for _, fun := range buses {
		a := fun()
		result[a.String()] = a
	}
	return result
}
