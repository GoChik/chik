package handlers

import (
	"encoding/json"
	"iosomething"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rferrazz/go-selfupdate/selfupdate"
)

// Current software version
var Version = "dev"

const configFile = "updates.json"

type configuration struct {
	UpdatesURL string
}

type updater struct {
	iosomething.BaseHandler
	updater      *selfupdate.Updater
	stopChannnel chan bool
}

// NewUpdater creates an updater from conf stored in config file.
// If conf file is not there it creates a default one searching for updates on the local machine
func NewUpdater(identity string) iosomething.Handler {
	conf := configuration{"http://127.0.0.1"}
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
		BaseHandler: iosomething.NewHandler(identity),
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

func (h *updater) checkerRoutine() {
	ticker := time.NewTicker(24*time.Hour + time.Duration(rand.Int63n(int64(60*time.Minute))))
	for {
		select {
		case <-h.stopChannnel:
			ticker.Stop()
			return

		case <-ticker.C:
			logrus.Debug("Periodically checking for updates")
			h.updater.FetchInfo()
		}
	}
}

func (h *updater) SetUp(remote chan<- *iosomething.Message) chan bool {
	logrus.Debug("Current version: ", Version)
	h.Remote = remote
	go h.checkerRoutine()
	return h.Error
}

func (h *updater) TearDown() {
	h.stopChannnel <- true
}

func (h *updater) handleRequestCommand(message *iosomething.Message) bool {
	requestCommand := iosomething.SimpleCommand{}
	err := json.Unmarshal(message.Data(), &requestCommand)
	if err != nil || requestCommand.Command != iosomething.GET_VERSION {
		return false
	}

	logrus.Debug("Getting update info from: ", h.updater.ApiURL)

	if h.updater.Info.Version == "" {
		logrus.Debug("Checking for updates")
		h.updater.FetchInfo()
	}

	data, err := json.Marshal(iosomething.VersionIndication{h.updater.CurrentVersion, h.updater.Info.Version})
	if err != nil {
		logrus.Error("Unable to marshal version message")
		h.Error <- true
	}

	sender, err := message.SenderUUID()
	if err != nil {
		logrus.Error("Cannot fetch message sender")
		return false
	}

	h.Remote <- iosomething.NewMessage(iosomething.MESSAGE, h.ID, sender, data)
	return true
}

func (h *updater) handleUpdateCommand(message *iosomething.Message) bool {
	updateCommand := iosomething.SimpleCommand{}
	err := json.Unmarshal(message.Data(), &updateCommand)
	if (err != nil) || (updateCommand.Command != iosomething.DO_UPDATE) {
		return false
	}

	logrus.Debug("Updating to version: ", h.updater.Info.Version)
	h.updater.BackgroundRun()

	// relaunch current executable
	args := os.Args[:]
	args[0] = h.updater.CmdName
	syscall.Exec(h.updater.CmdName, args, os.Environ())
	// and exit
	os.Exit(0)

	return true
}

func (h *updater) Handle(message *iosomething.Message) {
	switch {
	case h.handleRequestCommand(message):
		return

	case h.handleUpdateCommand(message):
		return
	}
}
