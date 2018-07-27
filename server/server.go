package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"iosomething"
	"iosomething/handlers"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

// CONFFILE configuration file
const CONFFILE = "server.json"

var peers = make(map[uuid.UUID]*iosomething.Remote)

type Configuration struct {
	Port        uint16
	PubKeyPath  string
	PrivKeyPath string
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	conffile := iosomething.GetConfPath(CONFFILE)

	if conffile == "" {
		logrus.Fatal("Config file not found")
	}

	fd, err := os.Open(conffile)
	if err != nil {
		logrus.Fatal("Cannot open config file", err)
	}
	defer fd.Close()

	decoder := json.NewDecoder(fd)
	conf := Configuration{}
	err = decoder.Decode(&conf)

	if err != nil {
		logrus.Fatal("Error reading config file", err)
	}

	logrus.Debug(conf)

	cert, err := tls.LoadX509KeyPair(conf.PubKeyPath, conf.PrivKeyPath)
	if err != nil {
		logrus.Fatal("Error loading tls certificate", err)
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", conf.Port), &config)
	if err != nil {
		logrus.Fatal("Error listening", err)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			logrus.Debug("Connection error", err)
			continue
		}

		// Creating the remote that is handling the newly connected client
		remote := iosomething.NewRemote(connection, 5*time.Minute)
		go handlers.NewForwardingHandler(peers).HandlerRoutine(remote)
		go handlers.NewHeartBeatHandler(uuid.NewV1(), 2*time.Minute).HandlerRoutine(remote)
	}
}
