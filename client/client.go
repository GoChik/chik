package main

import (
	"crypto/tls"
	"encoding/json"
	"iosomething/actuator"
	"iosomething/utils"
	"net"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
)

// CONFFILE configuration filename
const CONFFILE = "client.json"

func client(identity string, conn net.Conn) {
	remote := utils.NewRemote(conn)
	stop := remote.StopChannel()

	act := actuator.NewActuator()
	act.Initialize()
	defer act.Deinitialize()

	id, _ := uuid.FromString(identity)
	remote.OutBuffer <- utils.NewMessage(utils.MESSAGE, id, uuid.Nil, []byte{})

	for {
		select {
		case <-stop:
			logrus.Debug("Exiting client")
			return

		case message := <-remote.InBuffer:
			if message.Type() == utils.HEARTBEAT {
				logrus.Debug("Heartbeat received")
				continue
			}

			command := utils.DigitalCommand{}
			err := json.Unmarshal(message.Data(), &command)
			if err != nil {
				logrus.Error("Error parsing command", err)
				continue
			}

			go act.ExecuteCommand(&command)
		}
	}
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	path := utils.GetConfPath(CONFFILE)

	if path == "" {
		logrus.Fatal("Cannot find config file")
	}

	conf := utils.ClientConfiguration{}
	err := utils.ParseConf(path, &conf)
	if err != nil {
		logrus.Fatal("Error parsing config file", err)
	}

	if conf.Identity == "" {
		conf.Identity = uuid.NewV1().String()
		err = utils.UpdateConf(path, &conf)
		if err != nil {
			logrus.Fatal("Unable to update config file", err)
		}
	}

	logrus.Debug("Identity: ", conf.Identity)

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	for {
		logrus.Debug("Connecting to: ", conf.Server)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 1 * time.Minute}, "tcp", conf.Server, &tlsConf)
		if err == nil {
			client(conf.Identity, conn)
		}

		time.Sleep(10 * time.Second)
	}
}
