package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"io"
	"iosomething/actuator"
	"iosomething/utils"
	"log"
	"time"

	"github.com/satori/go.uuid"
)

// CONFFILE configuration filename
const CONFFILE = "client.json"

// Client starts a new client
func Client(identity string, conn *tls.Conn) {
	defer conn.Close()

	actuator.Initialize()
	defer actuator.Deinitialize()

	id, _ := uuid.FromString(identity)
	bytes := id.Bytes()

	binary.Write(conn, binary.BigEndian, uint8(utils.IDENTITY))
	binary.Write(conn, binary.BigEndian, uint32(16))
	binary.Write(conn, binary.BigEndian, bytes)

	reader := bufio.NewReader(conn)

	for {
		header, err := utils.ParseHeader(reader)
		if err != nil {
			log.Println("Header error", err)
			break
		}

		if header.MsgType == utils.HEARTBEAT {
			log.Println("Heartbeat received")
			continue
		}

		log.Println("Message header:", header)

		from := make([]byte, 16)
		_, err = io.ReadFull(reader, from)
		if err != nil {
			log.Println("Error parsing sender bytes")
			break
		}

		log.Println("Sender", from)

		message := make([]byte, header.MsgLen-16)
		_, err = io.ReadFull(reader, message)
		if err != nil {
			log.Println("Error reading message")
			break
		}

		log.Println("Message received")

		command := utils.DigitalCommand{}
		err = json.Unmarshal(message, &command)
		if err != nil {
			log.Println("Error parsing command", err)
		}

		go actuator.ExecuteCommand(&command)
	}
}

func main() {
	path := utils.GetConfPath(CONFFILE)

	if path == "" {
		log.Fatalln("Cannot find config file")
	}

	conf := utils.ClientConfiguration{}
	err := utils.ParseConf(path, &conf)

	if err != nil {
		log.Fatalln("Error parsing config file", err)
	}

	if conf.Identity == "" {
		conf.Identity = uuid.NewV1().String()
		err = utils.UpdateConf(path, &conf)
		if err != nil {
			log.Fatalln("Unable to update config file", err)
		}
	}

	log.Println("Identity: ", conf.Identity)

	tlsConf := tls.Config{
		InsecureSkipVerify: true,
	}

	for {
		log.Println("Connecting to: ", conf.Server)

		conn, err := tls.Dial("tcp", conf.Server, &tlsConf)
		if err == nil {
			Client(conf.Identity, conn)
		}

		time.Sleep(10 * time.Second)
	}
}
