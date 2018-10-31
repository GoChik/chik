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
		config.Sync()
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

func (h *updater) Run(remote *chik.Controller) {
	logrus.Debug("starting version handler")
	in := remote.PubSub.Sub(chik.VersionRequestCommandType.String())
	for data := range in {
		message := data.(*chik.Message)
		command := chik.SimpleCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Warn("Unexpected message")
			continue
		}

		if len(command.Command) != 1 {
			logrus.Error("Unexpected composed command")
			continue
		}

		switch command.Command[0] {
		case chik.GET:
			sender := message.SenderUUID()
			if sender == uuid.Nil {
				logrus.Warn("Cannot get sender")
				break
			}
			logrus.Debug("Getting update info from: ", h.updater.ApiURL)
			h.updater.FetchInfo()
			version := chik.VersionIndication{h.updater.CurrentVersion, h.updater.Info.Version}
			remote.Reply(message, chik.VersionReplyCommandType, version)

		case chik.SET:
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
