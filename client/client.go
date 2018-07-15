package main

import (
	"crypto/tls"
	"iosomething"
	"iosomething/handlers"
	"net"
	"sync"
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

	identity, err := uuid.FromString(conf.Identity)

	if err != nil {
		identity = uuid.NewV1()
		conf.Identity = identity.String()
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
	handlerList := []iosomething.Handler{
		handlers.NewIoHandler(identity),
		handlers.NewTimers(identity),
		handlers.NewHeartBeatHandler(identity, 2*time.Minute),
		handlers.NewUpdater(identity),
	}
	handlerList = append(handlerList, handlers.NewStatusHandler(identity, handlerList))

	// Listening network
	for {
		logrus.Debug("Connecting to: ", conf.Server)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", conf.Server, &tlsConf)
		if err == nil {
			logrus.Debug("New connection")
			remote := iosomething.NewRemote(conn, 10*time.Minute)
			wg := sync.WaitGroup{}
			wg.Add(len(handlerList))
			for _, h := range handlerList {
				go func(h iosomething.Handler) {
					defer wg.Done()
					h.HandlerRoutine(remote)
				}(h)
			}
			wg.Wait()
		}
		time.Sleep(10 * time.Second)
	}
}
