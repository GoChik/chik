package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"iosomething/utils"
	"log"
	"os"
	"strconv"
	"time"

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

// identity: | UUID |
func parseIdentity(reader *bufio.Reader, len uint32) (uid uuid.UUID, err error) {
	raw := make([]byte, 16)
	_, err = io.ReadFull(reader, raw)
	if err != nil {
		return
	}

	uid, _ = uuid.FromBytes(raw)
	return
}

// forwardMessage: | UUID | encrypted data |
func forwardMessage(from uuid.UUID, reader *bufio.Reader, length uint32) (err error) {
	reciver, err := parseIdentity(reader, 16)
	if err != nil {
		return
	}

	log.Println("Forwarding to", reciver)

	sender := clients[reciver]
	if sender == nil {
		err = errors.New("Cannot forward message, receiver disconnected")
		return
	}

	log.Println("Length", length)

	message := make([]byte, length-16)
	_, err = io.ReadFull(reader, message)
	if err != nil {
		return
	}

	err = sender.SendMessage(utils.MESSAGE, from, message)
	return
}

func Heartbeat(client *utils.Remote) error {
	for {
		time.Sleep(30 * time.Second)

		err := client.SendMessage(utils.HEARTBEAT, "")
		if err != nil {
			log.Println("Stopping heartbeat")
			return err
		}
	}
}

func Client(client *utils.Remote) {
	defer client.Conn.Close()

	reader := bufio.NewReader(client.Conn)

	header, err := utils.ParseHeader(reader)

	switch {
	case err != nil:
		log.Println("Error: type header cannot be read", err)
		return

	case header.MsgType != utils.IDENTITY:
		log.Println("Error: cannot identify client")
		return
	}

	identity, err := parseIdentity(reader, header.MsgLen)

	if err != nil {
		log.Println("Error: cannot parse client identity")
		return
	}

	log.Println("New client:", identity)

	clients[identity] = client

	for {
		header, err := utils.ParseHeader(reader)

		switch {
		case err != nil:
			log.Println("Error: type header cannot be read")
			break

		case header.MsgType != utils.MESSAGE:
			log.Println("Error: unexpected message")
			break
		}

		err = forwardMessage(identity, reader, header.MsgLen)
		if err != nil {
			log.Println("Message error", err)
			break
		}
	}

	log.Println("Disconnecting client", identity)
	delete(clients, identity)
}

func main() {
	conffile := utils.GetConfPath(CONFFILE)

	if conffile == "" {
		log.Fatalln("Config file not found")
	}

	fd, err := os.Open(conffile)
	defer fd.Close()

	if err != nil {
		log.Fatalln("Cannot open config file", err)
	}

	decoder := json.NewDecoder(fd)
	conf := Configuration{}
	err = decoder.Decode(&conf)

	if err != nil {
		log.Fatalln("Error reading config file", err)
	}

	log.Println(conf)

	cert, err := tls.LoadX509KeyPair(conf.PubKeyPath, conf.PrivKeyPath)

	if err != nil {
		log.Fatalln("Error loading tls certificate", err)
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", "0.0.0.0:"+strconv.Itoa(int(conf.Port)), &config)

	if err != nil {
		log.Fatalln("Error listening", err)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Println("Connection error", err)
			continue
		}

		client := utils.NewRemote(connection)

		go Heartbeat(&client)
		go Client(&client)
	}
}
