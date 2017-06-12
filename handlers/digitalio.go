package handlers

import (
	"iosomething"
	"iosomething/actuator"
	"time"

	"encoding/json"

	"math"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

type digitalHandler struct {
	iosomething.BaseHandler
	actuator    actuator.Actuator
	timers      []iosomething.DigitalCommand
	stopCounter chan bool
	lastTimer   uint16
}

type replyData struct {
	reply  []byte
	sender uuid.UUID
}

// NewDigitalIOHandler creates an handler that is working with the selected actuator
func NewDigitalIOHandler(identity string) iosomething.Handler {
	return &digitalHandler{
		iosomething.NewHandler(identity),
		actuator.NewActuator(),
		make([]iosomething.DigitalCommand, 0),
		make(chan bool, 1),
		1,
	}
}

func (h *digitalHandler) getTimers(pin int) []iosomething.TimedCommand {
	result := []iosomething.TimedCommand{}
	for _, timer := range h.timers {
		if timer.Pin == pin {
			result = append(result, iosomething.TimedCommand{
				timer.TimerID,
				timer.Command,
				timer.DelayMinutes,
			})
		}
	}

	return result
}

func (h *digitalHandler) execute(command iosomething.CommandType, pin int) []byte {
	switch command {
	case iosomething.SWITCH_OFF:
		logrus.Debug("Turning off pin ", pin)
		h.actuator.TurnOff(pin)
		break

	case iosomething.SWITCH_ON:
		logrus.Debug("Turning on pin ", pin)
		h.actuator.TurnOn(pin)
		break

	case iosomething.PUSH_BUTTON:
		logrus.Debug("Turning on and off pin ", pin)
		h.actuator.TurnOn(pin)
		time.Sleep(1 * time.Second)
		h.actuator.TurnOff(pin)
		break

	case iosomething.TOGGLE_ON_OFF:
		logrus.Debug("Switching pin ", pin)
		if h.actuator.GetStatus(pin) {
			h.actuator.TurnOff(pin)
		} else {
			h.actuator.TurnOn(pin)
		}
		break

	case iosomething.GET_STATUS:
		logrus.Debug("Status request for pin ", pin)
		reply, err := json.Marshal(iosomething.StatusIndication{
			Pin:    pin,
			Status: h.actuator.GetStatus(pin),
			Timers: h.getTimers(pin),
		})

		if err != nil {
			logrus.Error("Error formatting status reply")
			return nil
		}
		return reply
	}

	return nil
}

func (h *digitalHandler) startCounter() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			i := 0
			for _, timer := range h.timers {
				timer.DelayMinutes = timer.DelayMinutes - 1
				if timer.DelayMinutes <= 0 {
					h.execute(timer.Command, timer.Pin)
				} else {
					h.timers[i] = timer
					i++
				}
			}
			h.timers = h.timers[:i]
			break

		case <-h.stopCounter:
			return
		}
	}
}

func (h *digitalHandler) SetUp(remote chan<- *iosomething.Message) chan bool {
	h.Remote = remote
	h.actuator.Initialize()
	go h.startCounter()
	return h.Error
}

func (h *digitalHandler) TearDown() {
	h.actuator.Deinitialize()
	h.stopCounter <- true
}

func (h *digitalHandler) Handle(message *iosomething.Message) {
	receiver, err := message.ReceiverUUID()
	if message.Type() != iosomething.MESSAGE {
		return
	}

	if err != nil || receiver != h.ID {
		logrus.Warn("Message has been dispatched to the wrong receiver")
		return
	}

	command := iosomething.DigitalCommand{}
	err = json.Unmarshal(message.Data(), &command)
	if err != nil {
		logrus.Debug("DigitalIO unable to handle such message")
		return
	}

	if command.DelayMinutes > 0 {
		command.TimerID = h.lastTimer
		h.timers = append(h.timers, command)
		h.lastTimer = (h.lastTimer + 1) % math.MaxUint16
		if h.lastTimer == 0 {
			h.lastTimer = 1
		}
		return
	}

	reply := h.execute(command.Command, command.Pin)
	if reply != nil {
		sender, err := message.SenderUUID()
		if err != nil {
			logrus.Warning("Reply for no one: ", err)
			return
		}

		h.Remote <- iosomething.NewMessage(iosomething.MESSAGE, h.ID, sender, reply)
	}
}
