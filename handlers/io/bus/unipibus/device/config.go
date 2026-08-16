package device

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PinType uint8

const (
	DigitalInput PinType = iota
	DigitalOutput
	RelayOutput
	AnalogOutput
	PwmOutput
)

func (pin PinType) String() string {
	switch pin {
	case DigitalInput:
		return "di"
	case DigitalOutput, PwmOutput:
		return "do"
	case RelayOutput:
		return "ro"
	case AnalogOutput:
		return "ao"
	}
	return ""
}

type Configurer interface {
	Path(kind PinType, group, pin uint8) string
	ValueFilename(kind PinType) string
}

func GetConf() Configurer {
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return nil
	}
	release := strings.Split(string(version), " ")[2]
	major, err := strconv.Atoi(strings.Split(release, ".")[0])
	if err != nil || major < 6 {
		return &LegacyConfig{}
	}
	return &Config{}
}

type LegacyConfig struct {
}

func (c *LegacyConfig) Path(kind PinType, group, pin uint8) string {
	return fmt.Sprintf("/sys/devices/platform/unipi_plc/io_group%d/%s_%d_%02d", group, kind.String(), group, pin)
}

func (c *LegacyConfig) ValueFilename(kind PinType) string {
	return fmt.Sprintf("%s_value", kind.String())
}

type Config struct {
}

func (c *Config) Path(kind PinType, group, pin uint8) string {
	// find device
	root := ""
	prefix := fmt.Sprintf("/sys/devices/platform/unipi_plc/io_group%d/", group)
	filepath.WalkDir(prefix, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && strings.HasSuffix(d.Name(), kind.String()) {
			root = path
		}
		return nil
	})
	return fmt.Sprintf("%s/%s%d.%d", root, strings.ToUpper(kind.String()), group, pin)
}

func (c *Config) ValueFilename(kind PinType) string {
	return "value"
}
