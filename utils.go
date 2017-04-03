package iosomething

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ClientConfiguration contains infos about the client
type ClientConfiguration struct {
	Server   string
	Identity string
}

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

	path := filepath.Join("/etc/iosomething", filename)
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
