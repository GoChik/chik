package modbus

import (
	"fmt"
	"time"

	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/types"

	modbusapi "github.com/gochik/modbus"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
	funk "github.com/thoas/go-funk"
)

var logger = log.With().Str("handler", "io").Str("bus", "modbus").Logger()

const (
	pollTime = 100 * time.Millisecond

	commandSize = 2
)

// Config is the configuration structure used to setup the modbus BUS
type Config struct {
	SerialPort string
	BaudRate   int
	Devices    []*device
}

type device struct {
	Id            string
	Register      uint16
	BitNumber     uint8
	DeviceAddress uint8
	IsCoil        bool
	Type          bus.DeviceKind // supported only DigitalInput and DigitalOutput
	status        bool
	actions       chan<- *deviceAction
}

type deviceAction struct {
	id     string
	action bool
}

func (d *device) ID() string {
	return d.Id
}

func (d *device) Kind() bus.DeviceKind {
	return d.Type
}

func (d *device) Description() bus.DeviceDescription {
	return bus.DeviceDescription{
		ID:    d.Id,
		Kind:  d.Kind(),
		State: d.GetStatus(),
	}
}

func (d *device) TurnOn() {
	if d.Type == bus.DigitalInputDevice {
		log.Error().Msgf("can't turn on %v. It is a readonly device", d.ID())
		return
	}
	if d.status != true {
		d.actions <- &deviceAction{d.ID(), true}
		d.status = true
	}
}

func (d *device) TurnOff() {
	if d.Type == bus.DigitalInputDevice {
		log.Error().Msgf("can't turn off %v. It is a readonly device", d.ID())
		return
	}
	if d.status != false {
		d.actions <- &deviceAction{d.ID(), false}
		d.status = false
	}
}

func (d *device) Toggle() {
	if d.Type == bus.DigitalInputDevice {
		log.Error().Msgf("can't toggle %v. It is a readonly device", d.ID())
		return
	}
	d.status = !d.status
	d.actions <- &deviceAction{d.ID(), d.status}
}

func (d *device) GetStatus() bool {
	logger.Debug().Msgf("Get Status of %v", d.Id)
	return d.status
}

func (d *device) setStatus(status bool) bool {
	if d.status == status {
		return false
	}
	d.status = status
	return true
}

type modbus struct {
	client         modbusapi.Client
	handler        modbusapi.ClientHandler
	devices        map[string]*device
	pollingTimer   *time.Ticker
	deviceChanges  chan string
	devicesActions chan *deviceAction
}

// New creates a new modbus bus
func New() bus.Bus {
	return &modbus{
		devices:        make(map[string]*device),
		deviceChanges:  make(chan string, 0),
		devicesActions: make(chan *deviceAction, 1),
	}
}

func (b *modbus) Initialize(conf interface{}) {
	var c Config
	err := types.Decode(conf, &c)
	if err != nil {
		logger.Fatal().Msgf("Failed initializing bus: %v", err)
		return
	}
	for _, d := range c.Devices {
		b.devices[d.ID()] = d
		d.actions = b.devicesActions
	}
	if c.SerialPort == "" {
		logger.Error().Msg("Cannot open serial port: device not specified.")
		return
	}
	port, err := serial.OpenPort(&serial.Config{
		Name:        c.SerialPort,
		Baud:        c.BaudRate,
		Parity:      serial.ParityNone,
		ReadTimeout: 200 * time.Millisecond,
	})
	if err != nil {
		logger.Fatal().Msgf("Cannot open serial port %v: %v", c.SerialPort, err)
	}
	b.handler = modbusapi.NewRTUClientHandler(port, uint32(c.BaudRate))
	b.client = modbusapi.NewClient(b.handler)
	b.startModbusWorker()
}

func (b *modbus) Deinitialize() {
	b.pollingTimer.Stop()
	b.handler.Close()
}

func (b *modbus) Device(id string) (bus.Device, error) {
	device, ok := b.devices[id]
	if !ok {
		return nil, fmt.Errorf("No modbus device with ID: %s found", id)
	}
	return device, nil
}

func (b *modbus) DeviceIds() []string {
	return funk.Map(b.devices, func(k string, v *device) string {
		return k
	}).([]string)
}

func (b *modbus) DeviceChanges() <-chan string {
	return b.deviceChanges
}

func (b *modbus) String() string {
	return "modbus"
}

func (b *modbus) writeCoil(device *device, high bool) error {
	b.handler.SetSlave(device.DeviceAddress)
	value := uint16(0x0000)
	if high {
		value = 0xff00
	}
	reply, err := b.client.WriteSingleCoil(device.Register, value)
	logger.Debug().Msgf("Changed status of digital appliance, reply: %v", reply)
	return err
}

func (b *modbus) writeRegister(device *device, high bool) error {
	var command uint16
	for _, d := range b.devices {
		if d.Kind() == bus.DigitalOutputDevice &&
			d.DeviceAddress == device.DeviceAddress &&
			d.Register == device.Register && d.status {
			command = command | (0x0001 << d.BitNumber)
		}
	}
	if high {
		command = command | (0x0001 << device.BitNumber)
	} else {
		command = command & (0xffff ^ (0x0001 << device.BitNumber))
	}

	b.handler.SetSlave(device.DeviceAddress)
	reply, err := b.client.WriteSingleRegister(device.Register, command)
	logger.Debug().Msgf("Changed status of digital appliance, reply: %v", reply)
	return err
}

func (b *modbus) sendAction(action *deviceAction) {
	device := b.devices[action.id]
	var err error
	for i := 0; i < 5; i++ {
		if device.IsCoil {
			err = b.writeCoil(device, action.action)
		} else {
			err = b.writeRegister(device, action.action)
		}
		if err != nil {
			logger.Err(err).Msgf("Failed changing status of %v, attempt: %v", device.ID(), i+1)
		} else {
			break
		}
	}
}

type mbDeviceGroup struct {
	devices []*device
}

type mbDeviceGroupByRegisterAddress map[uint16]mbDeviceGroup
type mbDescriptionByAddress map[uint8]mbDeviceGroupByRegisterAddress

func (b *modbus) polledRegistersList() mbDescriptionByAddress {
	polledDevices := make(mbDescriptionByAddress)
	for _, d := range b.devices {
		if d.Kind() != bus.DigitalInputDevice || d.IsCoil {
			continue
		}
		if polledDevices[d.DeviceAddress] == nil {
			polledDevices[d.DeviceAddress] = make(mbDeviceGroupByRegisterAddress)
		}
		currentGroup := polledDevices[d.DeviceAddress][d.Register]
		if currentGroup.devices == nil {
			currentGroup.devices = make([]*device, 0, 1)
		}
		currentGroup.devices = append(currentGroup.devices, d)
		polledDevices[d.DeviceAddress][d.Register] = currentGroup
	}
	return polledDevices
}

func (b *modbus) startModbusWorker() {
	polledRegisters := b.polledRegistersList()
	b.pollingTimer = time.NewTicker(pollTime)
	go func() {
		for {
			select {
			case _, ok := <-b.pollingTimer.C:
				if !ok {
					return
				}
				b.queryDeviceChanges(polledRegisters)
			case action := <-b.devicesActions:
				b.sendAction(action)
			}
		}
	}()
}

func (b *modbus) queryDeviceChanges(polledDevices mbDescriptionByAddress) {
	for k, v := range polledDevices {
		b.handler.SetSlave(k)
		for registerAddress, groupData := range v {
			response, err := b.client.ReadHoldingRegisters(registerAddress, 1)
			if err != nil {
				logger.Error().Msgf("Failed to read device: %v, address: %v, error: %v", k, registerAddress, err)
				continue
			}
			if response == nil || len(response) != 2 {
				logger.Error().Msgf("Unexpected reply: %v", response)
				continue
			}
			for _, d := range groupData.devices {
				if d.setStatus((response[1-(d.BitNumber/8)] & (0x01 << (d.BitNumber % 8))) > 0) {
					b.deviceChanges <- d.ID()
				}
			}
		}
	}
}
