package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"iosomething/utils"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
)

// CONFFILE configuration file
const CONFFILE = "server.json"

type Configuration struct {
	Port        uint16
	PubKeyPath  string
	PrivKeyPath string
}

var clients = make(map[uuid.UUID]*utils.Remote)

func forwardMessage(message *utils.Message) error {
	logrus.Debug("Forwarding init")
	receiver, err := message.ReceiverUUID()
	if err != nil {
		return err
	}

	if receiver == uuid.Nil { // No reciver specified
		logrus.Debug("Forwarding stopped")
		return nil
	}

	receiverRemote := clients[receiver]
	if receiverRemote == nil {
		return fmt.Errorf("%v disconnected", receiver)
	}

	receiverRemote.OutBuffer <- message
	logrus.Debug("Forwarding done")
	return nil
}

func heartbeat(client *utils.Remote) {
	stop := client.StopChannel()
	for {
		select {
		case <-stop:
			logrus.Debug("Stopping heartbeat")
			return

		case <-time.After(30 * time.Second):
			client.OutBuffer <- utils.NewMessage(utils.HEARTBEAT, uuid.Nil, uuid.Nil, []byte{})
		}
	}
}

func clientConnection(client *utils.Remote) {
	stop := client.StopChannel()
	sender := uuid.Nil

	for {
		select {
		case <-stop:
			logrus.Debug("Disconnecting client ", sender)
			delete(clients, sender)
			return

		case message := <-client.InBuffer:
			logrus.Debug("Message received")

			msgtype := message.Type()
			if msgtype != utils.MESSAGE {
				logrus.Error("Unexpected message", msgtype)
				continue
			}

			newSender, err := message.SenderUUID()
			if err != nil {
				logrus.Error("Unable to read sender UUID", err)
				continue
			}

			if sender == uuid.Nil {
				logrus.Debug("New client:", newSender)
				sender = newSender
				clients[sender] = client
			}

			// forward message
			err = forwardMessage(message)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	conffile := utils.GetConfPath(CONFFILE)

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

		client := utils.NewRemote(connection, 0)

		go heartbeat(client)
		go clientConnection(client)
	}
}
