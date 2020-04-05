package bus

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	funk "github.com/thoas/go-funk"
)

const w1BusPollingInterval = 1 * time.Minute
const w1DevicePath = "sys/bus/w1/devices/w1_bus_master"
const ds18b20Template = "28-%s/w1_slave" // Only DS18B20 thermostat
var temperatureRegExp = regexp.MustCompile(".* t=([0-9]+)$")

type w1Device struct {
	Id       string
	DeviceID string
	value    float32
	file     *os.File
}

func (d *w1Device) initialize() (err error) {
	path := []string{
		w1DevicePath,
		fmt.Sprintf(ds18b20Template, d.DeviceID),
	}
	d.file, err = os.OpenFile(strings.Join(path, "/"), os.O_RDONLY, 0600)
	if err != nil {
		return
	}
	d.getCurrentValue()
	return
}

func (d *w1Device) getCurrentValue() {
	buffer := make([]byte, 128)
	d.file.Seek(0, 0)
	size, err := d.file.Read(buffer)
	if err != nil {
		logrus.Errorf("[w1] Failed reading device %s: %v", d.DeviceID, err)
		return
	}
	temperatureString := temperatureRegExp.FindString(string(buffer[:size-1]))
	l := len(temperatureString)
	if l == 0 {
		logrus.Error("[w1] Error while parsing temperature from sensor")
		return
	}
	// in order to avoid doing float operations we add the decimal dot in the proper position
	// and convert directly the string to float
	temperatureString = temperatureString[:l-3] + "." + temperatureString[l-3:]
	temperature, err := strconv.ParseFloat(temperatureString, 32)
	if err != nil {
		logrus.Errorf("[w1] Failed parsing temperature string: %v", err)
		return
	}
	d.value = float32(temperature)
}

func (d *w1Device) ID() string {
	return d.Id
}

func (d *w1Device) Kind() DeviceKind {
	return AnalogInputDevice
}

func (d *w1Device) Description() DeviceDescription {
	return DeviceDescription{
		ID:    d.Id,
		Kind:  AnalogInputDevice,
		State: d.value,
	}
}

func (d *w1Device) GetValue() float32 {
	return d.value
}

type w1Bus struct {
	devices             map[string]*w1Device
	deviceNotifications chan string
	timer               *time.Ticker
}

func (b *w1Bus) String() string {
	return "w1"
}

func (b *w1Bus) Initialize(config interface{}) {
	b.deviceNotifications = make(chan string, 1)
	var devices []*w1Device
	types.Decode(config, &devices)
	b.devices = make(map[string]*w1Device, len(devices))
	for _, device := range devices {
		device.initialize()
		b.devices[device.Id] = device
	}
	b.devices = funk.ToMap(devices, "Id").(map[string]*w1Device)
	b.timer = time.NewTicker(w1BusPollingInterval)
	go func() {
		for range b.timer.C {
			for id, device := range b.devices {
				oldValue := device.value
				device.getCurrentValue()
				if oldValue != device.value {
					b.deviceNotifications <- id
				}
			}
		}
		close(b.deviceNotifications)
	}()
}

func (b *w1Bus) Deinitialize() {
	b.timer.Stop()
}

func (b *w1Bus) Device(id string) (Device, error) {
	device, ok := b.devices[id]
	if !ok {
		return nil, fmt.Errorf("[W1Bus] No soft device with ID: %s found", id)
	}
	return device, nil
}

func (b *w1Bus) DeviceIds() []string {
	return funk.Map(b.devices, func(k string, v *w1Device) string {
		return k
	}).([]string)
}

func (b *w1Bus) DeviceChanges() <-chan string {
	return b.deviceNotifications
}
