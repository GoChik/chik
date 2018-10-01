package main

import (
	"chik"
	"chik/config"
	"chik/handlers"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

func main() {
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
			config.Sync()
		}
		logrus.Fatal("Config file not found: stub created")
	}

	identity := uuid.FromStringOrNil(config.Get("identity").(string))
	if identity == uuid.Nil {
		logrus.Fatal("Cannot get id from config")
	}

	server := config.Get("server").(string)
	if server == "" {
		logrus.Fatal("Cannot get server from config")
	}

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	// Creating handlers
	handlerList := []chik.Handler{
		handlers.NewIoHandler(),
		handlers.NewTimers(),
		handlers.NewHeartBeatHandler(2 * time.Minute),
		handlers.NewUpdater(),
	}
	handlerList = append(handlerList, handlers.NewStatusHandler(handlerList))

	// Listening network
	for {
		logrus.Debug("Identity: ", identity)
		logrus.Debug("Server: ", server)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", server, &tlsConf)
		if err == nil {
			logrus.Debug("New connection")
			remote := chik.NewRemote(identity, conn, 5*time.Minute)
			wg := sync.WaitGroup{}
			wg.Add(len(handlerList))
			for _, h := range handlerList {
				go func(h chik.Handler) {
					h.Run(remote)
					wg.Done()
				}(h)
			}
			wg.Wait()
		}
		time.Sleep(10 * time.Second)
	}
}
