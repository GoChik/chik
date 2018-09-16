package main

import (
	"crypto/tls"
	"chik"
	"chik/config"
	"chik/handlers"
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
			config.Set("id", uuid.NewV1())
			config.Set("server", "")
			config.Sync()
		}
		logrus.Fatal("Config file not found: stub created")
	}

	identity := config.Get("id").(uuid.UUID)
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
		handlers.NewIoHandler(identity),
		handlers.NewTimers(identity),
		handlers.NewHeartBeatHandler(identity, 2*time.Minute),
		handlers.NewUpdater(identity),
	}
	handlerList = append(handlerList, handlers.NewStatusHandler(identity, handlerList))

	// Listening network
	for {
		logrus.Debug("Identity: ", identity)
		logrus.Debug("Server: ", server)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", server, &tlsConf)
		if err == nil {
			logrus.Debug("New connection")
			remote := chik.NewRemote(conn, 5*time.Minute)
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
