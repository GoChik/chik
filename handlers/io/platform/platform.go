package platform

import (
	"github.com/gochik/chik/handlers/io/bus"
)

var busList = make([]func() bus.Bus, 0)

// CreateBuses creates the set of available actuators
func CreateBuses() map[string]bus.Bus {
	result := make(map[string]bus.Bus, len(busList))
	for _, fun := range busList {
		a := fun()
		result[a.String()] = a
	}
	return result
}

func initializeBus(busElements ...func() bus.Bus) {
	for _, bus := range busElements {
		busList = append(busList, bus)
	}
}
