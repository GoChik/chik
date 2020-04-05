// +build unipiDevice

package bus

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

const unipiDevicePath = "/sys/devices/platform/unipi_plc/io_group%d/%s_%d_%02d"
const unipiDeviceValue = "%s_value"
const pollSpeed = 50 * time.Millisecond

func init() {
	actuators = append(actuators, func() Bus {
		return &unipiBus{}
	})
}

type unipiPinType uint8

const (
	unipiDigitalInput unipiPinType = iota
	unipiDigitalOutput
	unipiRelayOutput
	unipiAnalogOutput
)

func (pin unipiPinType) String() string {
	switch pin {
	case unipiDigitalInput:
		return "di"
	case unipiDigitalOutput:
		return "do"
	case unipiRelayOutput:
		return "ro"
	case unipiAnalogOutput:
		return "ao"
	}
	return ""
}

type unipiDevice struct {
	Id     string
	Group  uint8
	Pin    uint8
	Type   unipiPinType
	status uint32
	file   *os.File
	buffer []byte
}

func (d *unipiDevice) path() string {
	return fmt.Sprintf(unipiDevicePath, d.Group, d.Type.String(), d.Group, d.Pin)
}

func (d *unipiDevice) initialize() (err error) {
	path := []string{d.path(), fmt.Sprintf(unipiDeviceValue, d.Type.String())}
	openMode := os.O_RDWR
	if d.Kind() == DigitalInputDevice {
		openMode = os.O_RDONLY
	}
	d.file, err = os.OpenFile(strings.Join(path, "/"), openMode, 0600)
	if err != nil {
		logrus.Error("Unipi device initialization failed: ", err)
		return
	}
	d.buffer = make([]byte, 128)
	return d.fetchStatus()
}

func (d *unipiDevice) fetchStatus() error {
	d.file.Seek(0, 0)
	size, err := d.file.Read(d.buffer)
	if err != nil {
		return fmt.Errorf("Unipi device: error reading sysfs interface %s: %v", d.Id, err)
	}
	status, err := strconv.ParseUint(string(d.buffer[0:size-1]), 10, 32)
	if err != nil {
		return fmt.Errorf("Unipi device: error parsing status: %s", string(d.buffer))
	}
	d.status = uint32(status)
	return nil
}

func (d *unipiDevice) ID() string {
	return d.Id
}

func (d *unipiDevice) Kind() DeviceKind {
	switch d.Type {
	case unipiDigitalInput:
		return DigitalInputDevice

	case unipiDigitalOutput:
	case unipiRelayOutput:
		return DigitalOutputDevice

	case unipiAnalogOutput:
		return AnalogOutputDevice
	}
	return DigitalInputDevice
}

func (d *unipiDevice) Description() DeviceDescription {
	var status interface{} = d.status
	if d.Type < unipiAnalogOutput {
		status = (d.status == 1)
	}
	return DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: status,
	}
}

func (d *unipiDevice) TurnOn() {
	d.file.Write([]byte{'1', '\n'})
	d.status = 1
}

func (d *unipiDevice) TurnOff() {
	d.file.Write([]byte{'0', '\n'})
	d.status = 0
}

func (d *unipiDevice) Toggle() {
	if d.status == 1 {
		d.TurnOff()
		return
	}
	d.TurnOn()
}

func (d *unipiDevice) GetStatus() bool {
	return d.status == 1
}

type unipiBus struct {
	devices             map[string]*unipiDevice
	polledDevices       []*unipiDevice
	deviceNotifications chan string
	notificationTimer   *time.Ticker
}

func (b *unipiBus) startPoll(frequency time.Duration) {
	b.deviceNotifications = make(chan string, 1)
	b.notificationTimer = time.NewTicker(frequency)
	go func() {
		for range b.notificationTimer.C {
			for _, device := range b.polledDevices {
				oldStatus := device.status
				err := device.fetchStatus()
				if err != nil {
					logrus.Error("Error fetching device status: ", err)
				}
				if oldStatus != device.status {
					b.deviceNotifications <- device.Id
				}
			}
		}
		close(b.deviceNotifications)
	}()
}

func (b *unipiBus) String() string {
	return "unipi"
}

func (b *unipiBus) Initialize(config interface{}) {
	logrus.Debug("Initialising unipi bus")
	var devices []*unipiDevice
	types.Decode(config, &devices)
	b.polledDevices = make([]*unipiDevice, 0)
	for _, device := range devices {
		device.initialize()
		if device.Type == unipiDigitalInput {
			logrus.Debugf("Add %s to polled devices", device.Id)
			b.polledDevices = append(b.polledDevices, device)
		}
	}
	b.devices = funk.ToMap(devices, "Id").(map[string]*unipiDevice)
	b.startPoll(pollSpeed)
}

func (b *unipiBus) Deinitialize() {
	b.notificationTimer.Stop()
	for _, v := range b.devices {
		v.file.Close()
	}
}

func (b *unipiBus) Device(id string) (Device, error) {
	device, ok := b.devices[id]
	if !ok {
		return nil, fmt.Errorf("[UnipiBus] No soft device with ID: %s found", id)
	}
	return device, nil
}

func (b *unipiBus) DeviceIds() []string {
	return funk.Map(b.devices, func(k string, v *unipiDevice) string {
		return k
	}).([]string)
}

func (b *unipiBus) DeviceChanges() <-chan string {
	return b.deviceNotifications
}
