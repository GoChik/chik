package main

import (
	"bufio"
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
	receiver, err := message.ReceiverUUID()
	if err != nil {
		return err
	}

	if receiver == uuid.Nil { // No reciver specified
		return nil
	}

	receiverRemote := clients[receiver]
	if receiverRemote == nil {
		return fmt.Errorf("%v disconnected", receiver)
	}

	return receiverRemote.SendMessage(message.Bytes())
}

func heartbeat(client *utils.Remote) error {
	for {
		time.Sleep(30 * time.Second)

		err := client.SendMessage(utils.NewMessage(utils.HEARTBEAT, uuid.Nil, uuid.Nil, []byte{}).Bytes())
		if err != nil {
			logrus.Debug("Stopping heartbeat")
			return err
		}
	}
}

func clientConnection(client *utils.Remote) {
	defer client.Terminate()

	reader := bufio.NewReader(client.Conn)
	var sender = uuid.Nil

	for {
		message, err := utils.ParseMessage(reader)
		if err != nil {
			logrus.Error("Error parsing message", err)
			break
		}

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

		if sender != newSender {
			logrus.Error("Sender uuid has changed")
			break
		}

		// forward message
		err = forwardMessage(message)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}

	logrus.Debug("Disconnecting client", sender)
	delete(clients, sender)
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

		client := utils.NewRemote(connection)

		go heartbeat(&client)
		go clientConnection(&client)
	}
}
