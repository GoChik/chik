package main

import (
	"crypto/tls"
	"iosomething"
	"iosomething/handlers"
	"iosomething/plugins"
	"net"
	"sort"
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

	conf := iosomething.ClientConfiguration{Plugins: []string{}}
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

	// Plugins initialization
	plugins := []plugins.Plugin{
		plugins.NewSystemTimePlugin(),
		plugins.NewWebServicePlugin(conf.Identity),
	}

	for _, p := range plugins {
		i := sort.SearchStrings(conf.Plugins, p.Name())
		if i < len(conf.Plugins) && conf.Plugins[i] == p.Name() {
			logrus.Debug("Starting plugin ", p.Name())
			p.Start()
			defer p.Stop()
		}
	}

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	// Creating handlers
	client := iosomething.NewListener([]iosomething.Handler{
		handlers.NewDigitalIOHandler(conf.Identity),
		handlers.NewHeartBeatHandler(conf.Identity, 2*time.Minute, false),
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

		time.Sleep(10 * time.Second)
	}
}
