package iosomething

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ClientConfiguration contains infos about the client
type ClientConfiguration struct {
	Server   string
	Identity string
}

const configPath = "/etc/iosomething"

// GetConfPath returns directory in which configuration file is
// returns an empty string if config file cannot be found
func GetConfPath(filename string) string {
	cwd, err := os.Getwd()

	if err == nil {
		path := filepath.Join(cwd, filename)
		_, err = os.Stat(path)

		if err == nil {
			return path
		}
	}

	path := filepath.Join(configPath, filename)
	_, err = os.Stat(path)
	if err == nil {
		return path
	}

	return ""
}

// ParseConf parse the config file
func ParseConf(path string, obj interface{}) error {
	fd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	decoder := json.NewDecoder(fd)
	err = decoder.Decode(obj)

	if err != nil {
		return err
	}

	return nil
}

// UpdateConf updates the config file with data from obj
func UpdateConf(path string, obj interface{}) error {
	os.Rename(path, path+".old")

	fd, err := os.Create(path)
	defer fd.Close()

	if err != nil {
		return err
	}

	fd.Chmod(os.ModeAppend)
	fd.Chmod(0644)

	data, err := json.MarshalIndent(obj, "", "  ")
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

// CreateConfigFile creates the physical config file with a provided default configuration
func CreateConfigFile(filename string, defaultConf interface{}) error {
	data, err := json.MarshalIndent(defaultConf, "", " ")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(cwd, filename)
	if _, err = os.Stat(path); err == nil {
		return errors.New("Config file already exists")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	file.Write(data)
	return nil
}
