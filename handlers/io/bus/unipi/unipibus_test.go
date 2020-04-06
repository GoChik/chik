package unipibus_test

import (
	"testing"
)

type dpath struct {
	device       unipiDevice
	expectedPath string
}

func TestString(t *testing.T) {
	for _, d := range []dpath{
		dpath{
			unipiDevice{
				Id:    "test1",
				Group: 1,
				Pin:   2,
				Type:  unipiDigitalInput,
			},
			"/sys/devices/platform/unipi_plc/io_group1/di_1_02",
		},
	} {
		actual := d.device.Path()
		if actual != d.expectedPath {
			t.Errorf("got: %v, expected: %v", actual, d.expectedPath)
		}
	}
}
