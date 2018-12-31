package main

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers"
	"github.com/sirupsen/logrus"
)

func main() {
	config.SetConfigFileName("client.conf")
	config.AddSearchPath("/etc/chik")
	config.AddSearchPath(".")
	err := config.ParseConfig()
	if err != nil {
		logrus.Warn("Failed parsing config file: ", err)
	}

	server := config.Get("server").(string)
	if server == "" {
		config.Set("server", "127.0.0.1:6767")
		config.Sync()
		logrus.Fatal("Cannot get server from config")
	}

	logrus.Debug("Server: ", server)
	controller := chik.NewController()

	// Creating handlers
	handlerList := []chik.Handler{
		handlers.NewStatusHandler(),
		handlers.NewIoHandler(),
		handlers.NewTimers(),
		handlers.NewSunset(),
		handlers.NewHeartBeatHandler(2 * time.Minute),
		handlers.NewUpdater(),
	}

	for _, h := range handlerList {
		controller.Start(h)
	}

	// Listening network
	for {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", server, &tls.Config{})
		if err == nil {
			logrus.Debug("New connection")
			<-controller.Connect(conn)
		}
		time.Sleep(10 * time.Second)
	}
}
