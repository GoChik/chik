package snapcast

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/gochik/chik/handlers/io/bus"
	"github.com/gochik/chik/handlers/snapcast"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
)

const MuteDevicePrefix = "snapcast_audio_mute"
const VolumeDevicePrefix = "snapcast_audio_volume"

var logger = log.With().Str("handler", "io").Str("bus", "snapcast").Logger()

type SnapcastDevice struct {
	id     string
	bus    *SnapcastBus
	volume snapcast.Volume
}

type SnapcastMuteDevice SnapcastDevice

func (d *SnapcastMuteDevice) ID() string {
	return fmt.Sprintf("%s_%s", MuteDevicePrefix, d.id)
}

func (d *SnapcastMuteDevice) Kind() bus.DeviceKind {
	return bus.DigitalOutputDevice
}

func (d *SnapcastMuteDevice) Description() bus.DeviceDescription {
	return bus.DeviceDescription{
		ID:    d.ID(),
		Kind:  d.Kind(),
		State: d.volume.Muted,
	}
}

func (d *SnapcastMuteDevice) TurnOn() {
	if d.volume.Muted {
		return
	}
	d.volume.Muted = true
	d.bus.SetClientVolume(d.id, &d.volume)
	d.bus.deviceChanges <- d.ID()
}

func (d *SnapcastMuteDevice) TurnOff() {
	if !d.volume.Muted {
		return
	}
	d.volume.Muted = false
	d.bus.SetClientVolume(d.id, &d.volume)
	d.bus.deviceChanges <- d.ID()
}

func (d *SnapcastMuteDevice) Toggle() {
	d.volume.Muted = !d.volume.Muted
	d.bus.SetClientVolume(d.id, &d.volume)
	d.bus.deviceChanges <- d.ID()
}

type SnapcastVolumeDevice SnapcastDevice

func (d *SnapcastVolumeDevice) ID() string {
	return fmt.Sprintf("%s_%s", VolumeDevicePrefix, d.id)
}

func (d *SnapcastVolumeDevice) Kind() bus.DeviceKind {
	return bus.AnalogOutputDevice
}

func (d *SnapcastVolumeDevice) Description() bus.DeviceDescription {
	return bus.DeviceDescription{
		ID:    d.ID(),
		Kind:  d.Kind(),
		State: d.volume.Percent,
	}
}

func (d *SnapcastVolumeDevice) SetValue(value float64) {
	intVal := int(math.Round(value))
	if intVal == d.volume.Percent {
		return
	}
	d.volume.Percent = intVal
	d.bus.SetClientVolume(d.id, &d.volume)
	d.bus.deviceChanges <- d.ID()
}

func (d *SnapcastVolumeDevice) AddValue(value float64) {
	d.volume.Percent += int(math.Round(value))
	d.bus.SetClientVolume(d.id, &d.volume)
	d.bus.deviceChanges <- d.ID()
}

type SnapcastConfig struct {
	SnapcastServerAddress string `json:"server_address" mapstructure:"server_address"`
}

type SnapcastBus struct {
	conf          SnapcastConfig
	devices       sync.Map
	deviceChanges chan string
	client        *jrpc2.Client
	lock          sync.Mutex
	stopLoop      context.CancelFunc
}

func New() bus.Bus {
	return &SnapcastBus{
		deviceChanges: make(chan string, 1),
		conf:          SnapcastConfig{SnapcastServerAddress: "127.0.0.1:1705"},
	}
}

func (b *SnapcastBus) String() string {
	return "snapcast"
}

func (b *SnapcastBus) request(method string, params interface{}) (*jrpc2.Response, error) {
	logger.Debug().Msg("Snapcast request begin")
	b.lock.Lock()
	defer b.lock.Unlock()
	logger.Debug().Msgf("Snapcast request: %s(%v)", method, params)
	if b.client == nil {
		err := b.connect()
		if err != nil {
			return nil, err
		}
	}
	for i := 0; i < 2; i += 1 {
		logger.Debug().Msgf("Tentative %v", i)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		resp, err := b.client.Call(ctx, method, params)
		if err != nil {
			logger.Err(err).Msg("Snapcast request failed")
			err = b.connect()
			if err != nil {
				return nil, err
			}
			continue
		}
		return resp, err
	}
	return nil, errors.New("Request failed")
}

func (b *SnapcastBus) GetClientVolume(clientID string) (snapcast.Volume, error) {
	resp, err := b.request("Client.GetStatus", map[string]interface{}{"id": clientID})
	logger.Debug().Msg("Status reply: ")
	if err != nil {
		logger.Err(err).Msg("Failed getting volume")
		return snapcast.Volume{}, err
	}
	var status snapcast.Client
	resp.UnmarshalResult(&status)
	logger.Debug().Msgf("Client status: %v", status)
	return status.Config.Volume, nil
}

func (b *SnapcastBus) SetClientVolume(clientID string, volume *snapcast.Volume) error {
	_, err := b.request("Client.SetVolume", map[string]interface{}{"id": clientID, "volume": *volume})
	return err
}

func (b *SnapcastBus) GetServerStatus() (status *snapcast.ServerStatus, err error) {
	var resp *jrpc2.Response
	resp, err = b.request("Server.GetStatus", nil)
	if err != nil {
		return
	}
	var fullStatus snapcast.Status
	err = resp.UnmarshalResult(&fullStatus)
	if err == nil {
		status = &fullStatus.Server
	}
	return
}

func (b *SnapcastBus) updateKnownClients(status *snapcast.ServerStatus) {
	for _, group := range status.Groups {
		for _, client := range group.Clients {
			updated := &SnapcastDevice{
				client.ID,
				b,
				client.Config.Volume,
			}
			_, loaded := b.devices.LoadOrStore(client.ID, updated)
			if !loaded {
				logger.Debug().Msgf("New device found %s", client.ID)
				b.deviceChanges <- (*SnapcastMuteDevice)(unsafe.Pointer(updated)).ID()
				b.deviceChanges <- (*SnapcastVolumeDevice)(unsafe.Pointer(updated)).ID()
			}
		}
	}
}

func (b *SnapcastBus) connect() error {
	if b.client != nil {
		b.client.Close()
		b.client = nil
	}

	conn, err := net.DialTimeout("tcp", b.conf.SnapcastServerAddress, 100*time.Millisecond)
	if err != nil {
		return err
	}

	b.client = jrpc2.NewClient(channel.Line(conn, conn), &jrpc2.ClientOptions{
		OnNotify: func(req *jrpc2.Request) {
			switch req.Method() {
			case "Client.OnVolumeChanged":
				var changed snapcast.ClientVolume
				err := req.UnmarshalParams(&changed)
				if err != nil {
					logger.Err(err).Msg("Failed decoding volume notification")
					return
				}
				updated := &SnapcastDevice{
					changed.ID,
					b,
					changed.Volume,
				}
				current, exists := b.devices.LoadOrStore(changed.ID, updated)
				if exists {
					current.(*SnapcastDevice).volume = updated.volume
				}
				b.deviceChanges <- (*SnapcastMuteDevice)(unsafe.Pointer(updated)).ID()
				b.deviceChanges <- (*SnapcastVolumeDevice)(unsafe.Pointer(updated)).ID()
				return
			case "Client.OnConnect", "Client.OnDisconnect":
				var changed snapcast.Client
				err := req.UnmarshalParams(&changed)
				if err != nil {
					logger.Err(err).Msg("Failed decoding client status notification")
					return
				}
				updated := &SnapcastDevice{
					changed.ID,
					b,
					changed.Config.Volume,
				}
				current, exists := b.devices.LoadOrStore(changed.ID, updated)
				if exists {
					current.(*SnapcastDevice).volume = changed.Config.Volume
				}
				b.deviceChanges <- (*SnapcastMuteDevice)(unsafe.Pointer(updated)).ID()
				b.deviceChanges <- (*SnapcastVolumeDevice)(unsafe.Pointer(updated)).ID()
				return
			}
		},
	})
	return nil
}

func (b *SnapcastBus) Initialize(config interface{}) {
	logger.Debug().Msgf("Snapcast config: %v", config)
	var c SnapcastConfig
	err := types.Decode(config, &c)
	if err == nil && c.SnapcastServerAddress != "" {
		b.conf = c
	}
	logger.Debug().Msgf("SNAPCAST CONFIG: %v", b.conf.SnapcastServerAddress)

	ctx, cancel := context.WithCancel(context.Background())
	b.stopLoop = cancel
	go func(ctx context.Context) {
		timeout := 5 * time.Second
		for {
			select {
			case <-ctx.Done():
				return

			case <-time.After(timeout):
				result, err := b.GetServerStatus()
				if err != nil {
					logger.Err(err).Msg("Failed getting server status")
					timeout = 5 * time.Second
				} else {
					b.DeviceIds()
					timeout = 5 * time.Minute
					b.updateKnownClients(result)
				}
			}
		}
	}(ctx)
}

// Deinitialize is used when actuator is going off
func (b *SnapcastBus) Deinitialize() {
	b.stopLoop()
	b.client.Close()
}

// Given an unique id returns the corresponding device, an error if the id does not correspond to a device
func (b *SnapcastBus) Device(id string) (bus.Device, error) {
	logger.Debug().Msgf("Device(%v)", id)
	splits := strings.Split(id, "_")
	if len(splits) < 3 {
		return nil, fmt.Errorf("Device not found")
	}
	prefix := strings.Join(splits[:3], "_")
	if prefix != MuteDevicePrefix && prefix != VolumeDevicePrefix {
		return nil, fmt.Errorf("Device not found")
	}
	rawID := strings.Join(splits[3:], "_")
	device, ok := b.devices.Load(rawID)
	if ok {
		d := device.(*SnapcastDevice)
		logger.Debug().Msgf("Device found %v", device)
		switch prefix {
		case MuteDevicePrefix:
			return (*SnapcastMuteDevice)(unsafe.Pointer(d)), nil
		case VolumeDevicePrefix:
			return (*SnapcastVolumeDevice)(unsafe.Pointer(d)), nil
		}
	}
	return nil, fmt.Errorf("Device not found")
}

// list of device ids this bus is handling
func (b *SnapcastBus) DeviceIds() []string {
	logger.Debug().Msg("DeviceIds")
	result := make([]string, 0)
	b.devices.Range(func(key, value interface{}) bool {
		d := value.(*SnapcastDevice)
		result = append(result,
			fmt.Sprintf("%s_%s", VolumeDevicePrefix, d.id),
			fmt.Sprintf("%s_%s", MuteDevicePrefix, d.id),
		)
		return true
	})
	logger.Debug().Strs("devices", result).Msg("replying with devices")
	return result
}

// Channel that returns the id of a device in the moment the device changes his status
func (b *SnapcastBus) DeviceChanges() <-chan string {
	return b.deviceChanges
}
