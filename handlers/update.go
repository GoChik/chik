package handlers

import (
	"encoding/json"
	"iosomething"
	"os"
	"syscall"

	"path/filepath"

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
	updater *selfupdate.Updater
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
		iosomething.NewHandler(identity),
		&selfupdate.Updater{
			CurrentVersion: Version,
			ApiURL:         conf.UpdatesURL,
			BinURL:         conf.UpdatesURL,
			DiffURL:        conf.UpdatesURL,
			Dir:            "/tmp",
			CmdName:        exe,
		}}
}

func (h *updater) SetUp(remote chan<- *iosomething.Message) chan bool {
	logrus.Debug("Current version: ", Version)
	h.Remote = remote
	return h.Error
}

func (h *updater) TearDown() {}

func (h *updater) handleRequestCommand(message *iosomething.Message) bool {
	requestCommand := iosomething.SimpleCommand{}
	err := json.Unmarshal(message.Data(), &requestCommand)
	if err != nil {
		return false
	}

	h.updater.FetchInfo()

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
