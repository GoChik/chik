package main

import (
	"chik"
	"chik/config"
	"chik/handlers"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
)

var peers = sync.Map{}

func main() {
	logrus.SetLevel(logrus.WarnLevel)
	config.SetConfigFileName("server.conf")
	config.AddSearchPath("/etc/chik")
	config.AddSearchPath(".")

	err := config.ParseConfig()
	if err != nil {
		if _, ok := err.(*config.FileNotFoundError); ok {
			id, _ := uuid.NewV1()
			config.Set("identity", id)
			config.Set("connection.port", 6767)
			config.Set("connection.public_key_path", "")
			config.Set("connection.private_key_path", "")
			config.Sync()
		}
		logrus.Fatal("Cannot parse config file: ", err)
	}

	identity := uuid.FromStringOrNil(config.Get("identity").(string))
	if identity == uuid.Nil {
		logrus.Fatal("Cannot get identity from config file")
	}

	publicKeyPath := config.Get("connection.public_key_path").(string)
	if publicKeyPath == "" {
		logrus.Fatal("Cannot get public key path from config file")
	}

	privateKeyPath := config.Get("connection.private_key_path").(string)
	if privateKeyPath == "" {
		logrus.Fatal("Cannot get private key path from config file")
	}

	port := config.Get("connection.port").(uint16)
	if port == 0 {
		logrus.Fatal("Cannot get port from config file")
	}

	cert, err := tls.LoadX509KeyPair(publicKeyPath, privateKeyPath)
	if err != nil {
		logrus.Fatal("Error loading tls certificate", err)
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port), &config)
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
		remote := chik.NewRemote(identity, connection, 5*time.Minute)
		go handlers.NewForwardingHandler(&peers).Run(remote)
		go handlers.NewHeartBeatHandler(2 * time.Minute).Run(remote)
	}
}
