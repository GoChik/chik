package main

import (
	"crypto/tls"
	"iosomething"
	"iosomething/handlers"
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
)

// CONFFILE configuration filename
const CONFFILE = "client.json"

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	// Configuration parsing
	path := iosomething.GetConfPath(CONFFILE)
	if path == "" {
		logrus.Fatal("Cannot find config file")
	}

	conf := iosomething.ClientConfiguration{}
	err := iosomething.ParseConf(path, &conf)
	if err != nil {
		logrus.Fatal("Error parsing config file", err)
	}

	if conf.Identity == "" {
		conf.Identity = uuid.NewV1().String()
		err = iosomething.UpdateConf(path, &conf)
		if err != nil {
			logrus.Fatal("Unable to update config file", err)
		}
	}

	logrus.Debug("Identity: ", conf.Identity)

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	// Creating handlers
	client := iosomething.NewListener([]iosomething.Handler{
		handlers.NewDigitalIOHandler(conf.Identity),
		handlers.NewHeartBeatHandler(conf.Identity, 2*time.Minute),
		handlers.NewUpdater(conf.Identity),
	})

	// Listening network
	for {
		logrus.Debug("Connecting to: ", conf.Server)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", conf.Server, &tlsConf)
		if err == nil {
			logrus.Debug("New connection")
			remote := iosomething.NewRemote(conn, 10*time.Minute)
			client.Listen(remote)
		}
	}
}
