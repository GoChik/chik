package handlers

import (
	"encoding/json"
	"iosomething"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
	"github.com/rferrazz/go-selfupdate/selfupdate"
)

// Current software version
var Version = "dev"

const configFile = "updates.json"

type configuration struct {
	UpdatesURL string
}

type updater struct {
	id      uuid.UUID
	updater *selfupdate.Updater
}

// NewUpdater creates an updater from conf stored in config file.
// If conf file is not there it creates a default one searching for updates on the local machine
func NewUpdater(identity uuid.UUID) iosomething.Handler {
	logrus.Debug("Current version: ", Version)
	conf := configuration{"http://dl.bintray.com/rferrazz/IO-Something"}
	confPath := iosomething.GetConfPath(configFile)
	if confPath == "" {
		if iosomething.CreateConfigFile(configFile, conf) != nil {
			logrus.Error("Cannot write updater configuration file")
		}
		confPath = iosomething.GetConfPath(configFile)
	}
	err := iosomething.ParseConf(confPath, &conf)
	if err != nil {
		logrus.Error("Cannot parse configuration")
	}

	executable, _ := os.Executable()
	exe := filepath.Base(executable)

	return &updater{
		id: identity,
		updater: &selfupdate.Updater{
			CurrentVersion: Version,
			ApiURL:         conf.UpdatesURL,
			BinURL:         conf.UpdatesURL,
			DiffURL:        conf.UpdatesURL,
			Dir:            "/tmp",
			CmdName:        exe,
		},
	}
}

func (h *updater) fetchVersion() {
	if h.updater.Info.Version == "" {
		logrus.Debug("Checking for updates")
		h.updater.FetchInfo()
	}
}

func (h *updater) handleRequestCommand(command *iosomething.SimpleCommand, sender uuid.UUID) *iosomething.Message {
	logrus.Debug("Getting update info from: ", h.updater.ApiURL)

	h.fetchVersion()
	data, err := json.Marshal(iosomething.VersionIndication{h.updater.CurrentVersion, h.updater.Info.Version})
	if err != nil {
		return nil
	}
	return iosomething.NewMessage(iosomething.VersionIndicationType, h.id, sender, data)
}

func (h *updater) update() {
	logrus.Debug("Updating to version: ", h.updater.Info.Version)
	h.updater.BackgroundRun()

	// relaunch current executable
	args := os.Args[:]
	args[0] = h.updater.CmdName
	syscall.Exec(h.updater.CmdName, args, os.Environ())
	// and exit
	os.Exit(0)
}

func (h *updater) HandlerRoutine(remote *iosomething.Remote) {
	logrus.Debug("starting version handler")
	in := remote.PubSub.Sub(iosomething.SimpleCommandType.String())
	for data := range in {
		message := data.(*iosomething.Message)
		command := iosomething.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Warn("Unexpected message")
			continue
		}

		switch command.Command {
		case iosomething.GET_VERSION:
			sender, err := message.SenderUUID()
			if err != nil {
				logrus.Warn("Cannot get sender")
				break
			}
			reply := h.handleRequestCommand(&command, sender)
			if reply != nil {
				remote.PubSub.Pub(reply, "out")
			}

		case iosomething.DO_UPDATE:
			h.update()
		}
	}
	logrus.Debug("shutting down version handler")
}

func (h *updater) Status() interface{} {
	h.fetchVersion()
	return map[string]interface{}{
		"current": h.updater.CurrentVersion,
		"latest":  h.updater.Info.Version,
	}
}

func (h *updater) String() string {
	return "version"
}
