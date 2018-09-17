package handlers

import (
	"chik"
	"chik/config"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
	"github.com/rferrazz/go-selfupdate/selfupdate"
)

// Current software version
var Version = "dev"

type updater struct {
	updater *selfupdate.Updater
}

// NewUpdater creates an updater from conf stored in config file.
// If conf file is not there it creates a default one searching for updates on the local machine
func NewUpdater() chik.Handler {
	logrus.Debug("Version: ", Version)
	updatesURL := "http://dl.bintray.com/rferrazz/IO-Something"
	value := config.Get("updater.url")
	if value == nil {
		config.Set("updater.url", updatesURL)
	} else {
		updatesURL = value.(string)
	}

	executable, _ := os.Executable()
	exe := filepath.Base(executable)

	return &updater{
		updater: &selfupdate.Updater{
			CurrentVersion: Version,
			ApiURL:         updatesURL,
			BinURL:         updatesURL,
			DiffURL:        updatesURL,
			Dir:            "/tmp",
			CmdName:        exe,
		},
	}
}

func (h *updater) handleRequestCommand(command *chik.SimpleCommand, sender uuid.UUID) *chik.Message {
	logrus.Debug("Getting update info from: ", h.updater.ApiURL)

	h.updater.FetchInfo()
	data, err := json.Marshal(chik.VersionIndication{h.updater.CurrentVersion, h.updater.Info.Version})
	if err != nil {
		return nil
	}
	return chik.NewMessage(chik.VersionIndicationType, uuid.Nil, sender, data)
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

func (h *updater) Run(remote *chik.Remote) {
	logrus.Debug("starting version handler")
	in := remote.PubSub.Sub(chik.SimpleCommandType.String())
	for data := range in {
		message := data.(*chik.Message)
		command := chik.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Warn("Unexpected message")
			continue
		}

		switch command.Command {
		case chik.GET_VERSION:
			sender, err := message.SenderUUID()
			if err != nil {
				logrus.Warn("Cannot get sender")
				break
			}
			reply := h.handleRequestCommand(&command, sender)
			if reply != nil {
				remote.PubSub.Pub(reply, "out")
			}

		case chik.DO_UPDATE:
			h.update()
		}
	}
	logrus.Debug("shutting down version handler")
}

func (h *updater) Status() interface{} {
	if h.updater.Info.Version == "" {
		h.updater.FetchInfo()
	}
	return map[string]interface{}{
		"current": h.updater.CurrentVersion,
		"latest":  h.updater.Info.Version,
	}
}

func (h *updater) String() string {
	return "version"
}
