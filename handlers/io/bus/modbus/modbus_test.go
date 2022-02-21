package modbus

import (
	"os"
	"testing"

	"github.com/gochik/chik/handlers/io/bus"
)

func TestInitialization(t *testing.T) {
	serialPort := os.Getenv("SERIAL_PORT")
	if serialPort == "" {
		t.Skip("Skipping because env var SERIAL_PORT is not set")
	}
	conf := Config{
		SerialPort: serialPort,
		BaudRate:   115200,
		Devices: []*device{
			{
				Id:            "test",
				Register:      0,
				BitNumber:     0,
				DeviceAddress: 1,
				Type:          bus.DigitalInputDevice,
			},
		},
	}
	bus := New()
	bus.Initialize(conf)
}
