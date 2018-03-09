package handlers

import (
	"iosomething"
	"iosomething/actuator"
	"time"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
	"github.com/thoas/go-funk"
)

type timeredCommand struct {
	command *iosomething.DigitalCommand
	timer   *time.Timer
}

type digitalHandler struct {
	iosomething.BaseHandler
	actuator  actuator.Actuator
	timers    []timeredCommand
	lastTimer uint16
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
		make([]timeredCommand, 0),
		1,
	}
}

func calculateNextAlarmTime(alarm iosomething.JSONTime) time.Time {
	now := time.Now()
	then := time.Date(now.Year(), now.Month(), now.Day(),
		time.Time(alarm).Hour(), time.Time(alarm).Minute(),
		0, 0, now.Location())
	if then.Sub(now) < 0 {
		then = then.AddDate(0, 0, 1)
	}
	return then
}

func (h *digitalHandler) execTimeredCommand(timer *timeredCommand) {
	<-timer.timer.C

	now := time.Now()
	then := calculateNextAlarmTime(timer.command.Time)
	currentDay := iosomething.EnabledDays(now.Day() + 1)

	if timer.command.Repeat == 0 || timer.command.Repeat&currentDay != 0 {
		h.execute(timer.command.Command, timer.command.Pin)
	}
	if timer.command.Repeat != 0 {
		timer.timer.Reset(then.Sub(now))
		go h.execTimeredCommand(timer)
	} else {
		h.deleteTimer(timer.command.TimerID)
	}
}

func (h *digitalHandler) addTimer(command *iosomething.DigitalCommand) {
	logrus.Debugf("Setting timer with time: %d:%d and repeat: %d",
		time.Time(command.Time).Hour(), time.Time(command.Time).Minute(), command.Repeat)

	command.TimerID = h.lastTimer
	h.lastTimer++
	then := calculateNextAlarmTime(command.Time)

	timer := timeredCommand{command, time.NewTimer(time.Until(then))}
	go h.execTimeredCommand(&timer)
	h.timers = append(h.timers, timer)
}

func (h *digitalHandler) getTimers(pin int) []iosomething.TimedCommand {
	result := []iosomething.TimedCommand{}
	for _, timer := range h.timers {
		if timer.command.Pin == pin {
			result = append(result, iosomething.TimedCommand{
				TimerID: timer.command.TimerID,
				Command: timer.command.Command,
				Time:    timer.command.Time,
				Repeat:  timer.command.Repeat,
			})
		}
	}

	return result
}

func (h *digitalHandler) deleteTimer(timerID uint16) {
	logrus.Debug("Deleting timer ", timerID)

	h.timers = funk.Filter(h.timers, func(t timeredCommand) bool {
		if t.command.TimerID == timerID {
			t.timer.Stop()
			return false
		}
		return true
	}).([]timeredCommand)
}

func (h *digitalHandler) editTimer(timer *iosomething.DigitalCommand) {
	logrus.Debug("Altering timer ", timer.TimerID)

	for index, storedTimer := range h.timers {
		if storedTimer.command.TimerID == timer.TimerID {
			storedTimer.timer.Reset(time.Until(calculateNextAlarmTime(timer.Time)))
			h.timers[index].command = timer
			break
		}
	}
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

func (h *digitalHandler) SetUp(remote chan<- *iosomething.Message) chan bool {
	h.Remote = remote
	h.actuator.Initialize()
	return h.Error
}

func (h *digitalHandler) TearDown() {
	for _, timer := range h.timers {
		timer.timer.Stop()
	}
	h.actuator.Deinitialize()
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
		return
	}

	switch {
	case command.Command == iosomething.DELETE_TIMER:
		h.deleteTimer(command.TimerID)
		break

	case command.TimerID == 0 && !time.Time(command.Time).IsZero():
		h.addTimer(&command)
		break

	case command.TimerID != 0:
		h.editTimer(&command)
		break

	default:
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
}
