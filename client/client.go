package main

import (
	"chik"
	"chik/config"
	"chik/handlers"
	"crypto/tls"
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})
	logrus.SetLevel(logrus.WarnLevel)

	// Config stuff
	config.SetConfigFileName("client.conf")
	config.AddSearchPath("/etc/chik")
	config.AddSearchPath(".")
	err := config.ParseConfig()

	if err != nil {
		if _, ok := err.(*config.FileNotFoundError); ok {
			id, _ := uuid.NewV1()
			config.Set("identity", id)
			config.Set("server", "")
			config.Set("log_level", "warning")
			config.Sync()
		}
		logrus.Fatal("Config file not found: stub created")
	}

	logLevel, err := logrus.ParseLevel(config.Get("log_level").(string))
	if err != nil {
		logrus.Fatal("Cannot set log level: ", err)
	}
	logrus.SetLevel(logLevel)

	identity := uuid.FromStringOrNil(config.Get("identity").(string))
	if identity == uuid.Nil {
		logrus.Fatal("Cannot get id from config")
	}

	server := config.Get("server").(string)
	if server == "" {
		logrus.Fatal("Cannot get server from config")
	}

	logrus.Debug("Identity: ", identity)
	logrus.Debug("Server: ", server)
	controller := chik.NewController(identity)

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
