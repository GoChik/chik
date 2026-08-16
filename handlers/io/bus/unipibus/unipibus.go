package unipibus

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/io/bus/unipibus/device"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	funk "github.com/thoas/go-funk"
)

var logger = log.With().Str("handler", "io").Str("bus", "unipi").Logger()

const pwmDutyCycle = "pwm_duty_cycle"
const pwmPrescale = "pwm_prescale"
const defaultPwmPrescale = 800
const pwmFrequencyCycle = "pwm_frequency_cycle"
const defaultPwmFrequencyCycle = 200
const pollSpeed = 50 * time.Millisecond

type unipiDevice struct {
	Id        string
	Group     uint8
	Pin       uint8
	Type      device.PinType
	status    int
	file      *os.File
	buffer    []byte
	fadeMutex sync.Mutex
}

func (d *unipiDevice) writeDefaults(filepath, value string) {
	os.WriteFile(
		filepath,
		[]byte(value),
		0644,
	)
}

func (d *unipiDevice) initialize(conf device.Configurer) (err error) {
	filename := conf.ValueFilename(d.Type)
	if d.Type == device.PwmOutput {
		filename = pwmDutyCycle
		d.writeDefaults(path.Join(conf.Path(d.Type, d.Group, d.Pin), pwmPrescale), strconv.Itoa(defaultPwmPrescale))
		d.writeDefaults(path.Join(conf.Path(d.Type, d.Group, d.Pin), pwmFrequencyCycle), strconv.Itoa(defaultPwmFrequencyCycle))
	}
	path := []string{conf.Path(d.Type, d.Group, d.Pin), filename}
	openMode := os.O_RDWR
	if d.Kind() == bus.DigitalInputDevice {
		openMode = os.O_RDONLY
	}
	d.file, err = os.OpenFile(strings.Join(path, "/"), openMode, 0600)
	if err != nil {
		logger.Error().Msgf("Device initialization failed: %v", err)
		return
	}
	d.buffer = make([]byte, 128)
	return d.fetchStatus()
}

func (d *unipiDevice) fetchStatus() error {
	d.file.Seek(0, 0)
	size, err := d.file.Read(d.buffer)
	if err != nil {
		return fmt.Errorf("failed reading sysfs interface %s: %v", d.Id, err)
	}
	status, err := strconv.ParseInt(string(d.buffer[0:size-1]), 10, 32)
	if err != nil {
		return fmt.Errorf("failed parsing status: %s", string(d.buffer))
	}
	d.status = int(status)
	return nil
}

func (d *unipiDevice) ID() string {
	return d.Id
}

func (d *unipiDevice) Kind() bus.DeviceKind {
	switch d.Type {
	case device.DigitalInput:
		return bus.DigitalInputDevice

	case device.DigitalOutput, device.RelayOutput:
		return bus.DigitalOutputDevice

	case device.AnalogOutput, device.PwmOutput:
		return bus.AnalogOutputDevice
	}
	return bus.DigitalInputDevice
}

func (d *unipiDevice) Description() bus.DeviceDescription {
	var status interface{} = d.status
	if d.Type < device.AnalogOutput {
		status = (d.status == 1)
	}
	return bus.DeviceDescription{
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

func (d *unipiDevice) pwmFade(from, to int) {
	d.fadeMutex.Lock()
	defer d.fadeMutex.Unlock()

	diff := to - from
	absDiff := int(math.Abs(float64(diff)))
	if absDiff == 0 {
		return
	}

	for i := 1; i < absDiff+1; i++ {
		d.file.Write([]byte(strconv.Itoa(from+(i*(absDiff/diff))) + "\n"))
		time.Sleep(20 * time.Millisecond)
	}
}

func (d *unipiDevice) SetValue(value float64) {
	if d.Type < device.AnalogOutput {
		logger.Error().Msgf("Cannot set analog value to device %v", d)
	}
	status := int(math.Round(value))
	if d.Type == device.PwmOutput {
		go d.pwmFade(d.status, status)
	} else {
		d.file.Write([]byte(strconv.Itoa(status) + "\n"))
	}
	d.status = status
}

func (d *unipiDevice) AddValue(value float64) {
	if d.Type < device.AnalogOutput {
		logger.Error().Msgf("Cannot set analog value to device %v", d)
	}
	status := int(math.Max(0, float64(d.status+int(math.Round(value)))))
	if d.Type == device.PwmOutput {
		go d.pwmFade(d.status, status)
	} else {
		d.file.Write([]byte(strconv.Itoa(status) + "\n"))
	}
	d.status = status
}

type unipiBus struct {
	devices             map[string]*unipiDevice
	polledDevices       []*unipiDevice
	deviceNotifications chan string
	notificationTimer   *time.Ticker
}

func New() bus.Bus {
	return &unipiBus{}
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
					logger.Error().Msgf("Error fetching device status: %v", err)
					continue
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
	logger.Debug().Msg("Initialising bus")
	conf := device.GetConf()
	var devices []*unipiDevice
	err := types.Decode(config, &devices)
	if err != nil {
		logger.Error().Msgf("Failed initializing bus: %v", err)
	}
	b.polledDevices = make([]*unipiDevice, 0)
	for _, d := range devices {
		d.initialize(conf)
		if d.Type == device.DigitalInput {
			logger.Debug().Msgf("Add %s to polled devices", d.Id)
			b.polledDevices = append(b.polledDevices, d)
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

func (b *unipiBus) Device(id string) (bus.Device, error) {
	device, ok := b.devices[id]
	if !ok {
		return nil, fmt.Errorf("no soft device with ID: %s found", id)
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
