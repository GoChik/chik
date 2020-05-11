package version

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/gofrs/uuid"
	"github.com/rferrazz/go-selfupdate/selfupdate"
	"github.com/sirupsen/logrus"
)

type updater struct {
	updater *selfupdate.Updater
}

// New creates an updater from conf stored in config file.
// If conf file is not there it creates a default one searching for updates on the local machine
func New(currentVersion string) chik.Handler {
	logrus.Debug("Version: ", currentVersion)
	updatesURL := "http://dl.bintray.com/rferrazz/IO-Something/"
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
			CurrentVersion: currentVersion,
			ApiURL:         updatesURL,
			BinURL:         updatesURL,
			DiffURL:        updatesURL,
			CmdName:        exe,
		},
	}
}

func (h *updater) update() {
	logrus.Debug("Updating to version: ", h.updater.Info.Version)
	h.updater.Apply()

	// TODO: launch update script and exit
	command := exec.Command("/sbin/reboot")
	command.Run()
	os.Exit(0)
}

func (h *updater) Dependencies() []string {
	return []string{}
}

func (h *updater) Topics() []types.CommandType {
	return []types.CommandType{types.VersionRequestCommandType}
}

func (h *updater) Setup(controller *chik.Controller) chik.Timer {
	// TODO: check periodically for new versions
	return chik.NewEmptyTimer()
}

func (h *updater) HandleMessage(message *chik.Message, remote *chik.Controller) {
	var command types.SimpleCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logrus.Warn("Unexpected message")
		return
	}

	switch command.Action {
	case types.GET:
		sender := message.SenderUUID()
		if sender == uuid.Nil {
			logrus.Warn("Cannot get sender")
			return
		}
		logrus.Debug("Getting update info from: ", h.updater.ApiURL)
		err := h.updater.FetchInfo()
		if err != nil {
			logrus.Warning("Cannot fetch update info:", err)
			return
		}
		version := types.VersionIndication{h.updater.CurrentVersion, h.updater.Info.Version}
		remote.Reply(message, types.VersionReplyCommandType, version)

	case types.SET:
		h.update()
	}
}

func (h *updater) HandleTimerEvent(tick time.Time, controller *chik.Controller) {}

func (h *updater) Teardown() {}

// TODO: remove and replace with status holder (updated on regular intervals)
func (h *updater) status() interface{} {
	if h.updater.Info.Version == "" {
		h.updater.FetchInfo()
	}
	return types.VersionIndication{
		h.updater.CurrentVersion,
		h.updater.Info.Version,
	}
}

func (h *updater) String() string {
	return "version"
}
