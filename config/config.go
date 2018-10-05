package config

import (
	"chik"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
)

var conf config
var once = sync.Once{}

type config struct {
	sync.Mutex
	searchPaths []string
	currentPath string
	fileName    string
	data        map[string]interface{}
}

// FileNotFoundError defines a config file not found
type FileNotFoundError struct {
}

func (*FileNotFoundError) Error() string {
	return "Config file not found in any of the search paths"
}

func new() {
	conf = config{
		searchPaths: make([]string, 0),
		currentPath: "",
		fileName:    "config",
		data:        make(map[string]interface{}),
	}
}

func init() {
	once.Do(func() {
		new()
	})
}

func parse(path string) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	decoder := json.NewDecoder(fd)
	err = decoder.Decode(&conf.data)

	if err != nil {
		return err
	}

	return nil
}

// AddSearchPath adds a path to the list of folders scanned in order to search the config file.
// When opening the config file paths are scanned the order they are added
func AddSearchPath(path string) error {
	// Get absolute path
	if filepath.IsAbs(path) {
		path = filepath.Clean(path)
	} else {
		path, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		path = filepath.Clean(path)
	}

	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	conf.Lock()
	conf.searchPaths = append(conf.searchPaths, path)
	conf.Unlock()
	return nil
}

// SetConfigFileName sets the configuration file name. The default value is "config"
func SetConfigFileName(name string) {
	conf.Lock()
	conf.fileName = name
	conf.Unlock()
}

// ParseConfig reads the config file
func ParseConfig() error {
	conf.Lock()
	defer conf.Unlock()

	conf.currentPath = ""
	for _, path := range conf.searchPaths {
		fullPath := filepath.Join(path, conf.fileName)
		_, err := os.Stat(fullPath)
		if err == nil {
			conf.currentPath = fullPath
		}
	}

	if conf.currentPath == "" {
		return &FileNotFoundError{}
	}

	return parse(conf.currentPath)
}

// Get returns a value from the current config
func Get(key string) interface{} {
	conf.Lock()
	defer conf.Unlock()

	slices := strings.Split(key, ".")
	sector := conf.data
	for i, slice := range slices {
		v := sector[slice]
		if v == nil {
			return nil
		}
		vType := reflect.TypeOf(v)
		if i == len(slices)-1 {
			return v
		}
		if vType.Kind() == reflect.Map {
			sector = v.(map[string]interface{})
		} else {
			return nil
		}
	}
	return nil
}

// GetStruct populates data of the given struct with config file content
func GetStruct(key string, output interface{}) error {
	data := Get(key)
	config := mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(chik.IntToJsIntArr, chik.StringToJsonTime),
		Result:           output,
	}
	decoder, err := mapstructure.NewDecoder(&config)
	if err != nil {
		return err
	}
	return decoder.Decode(data)
}

// Set sets or modifies a value in the config file.
func Set(key string, value interface{}) error {
	conf.Lock()
	defer conf.Unlock()

	slices := strings.Split(key, ".")
	v := conf.data
	for i, k := range slices {
		if i == len(slices)-1 {
			v[k] = value
			return nil
		}
		if v[k] == nil {
			v[k] = make(map[string]interface{})
		}
		v = v[k].(map[string]interface{})
	}
	return nil
}

// Sync writes the config back to file
func Sync() error {
	conf.Lock()
	defer conf.Unlock()

	if conf.currentPath == "" {
		if len(conf.searchPaths) > 0 {
			conf.currentPath = filepath.Join(conf.searchPaths[0], conf.fileName)
		} else {
			return errors.New("Unable to set a config path")
		}
	} else {
		os.Rename(conf.currentPath, conf.currentPath+".old")
	}

	fd, err := os.Create(conf.currentPath)
	defer fd.Close()

	if err != nil {
		return err
	}

	fd.Chmod(os.ModeAppend)
	fd.Chmod(0644)

	data, err := json.MarshalIndent(conf.data, "", "  ")
	if err != nil {
		return err
	}

	_, err = fd.Write(data)
	if err != nil {
		return err
	}

	fd.Sync()
	return nil
}
