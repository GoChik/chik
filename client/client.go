package main

import (
	"crypto/tls"
	"iosomething"
	"iosomething/handlers"
	"net"
	"time"

	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/beevik/ntp"
	"github.com/satori/go.uuid"
)

// CONFFILE configuration filename
const CONFFILE = "client.json"

func setSystemTime() {
	go func() {
		for {
			date, err := ntp.Time("0.pool.ntp.org")
			if err == nil {
				cmd := exec.Command("date", "-s", date.Format("2006.01.02-15:04:05"))
				cmd.Run()
			}
			time.Sleep(24 * 7 * time.Hour)
		}
	}()
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	path := iosomething.GetConfPath(CONFFILE)

	if path == "" {
		logrus.Fatal("Cannot find config file")
	}

	setSystemTime()

	conf := iosomething.ClientConfiguration{}
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

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	client := iosomething.NewListener([]iosomething.Handler{
		handlers.NewActuatorHandler(conf.Identity),
		handlers.NewHeartBeatHandler(conf.Identity, 2*time.Minute, false),
	})

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
