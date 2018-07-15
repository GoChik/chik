package handlers

import (
	"encoding/json"
	"iosomething"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"
	"github.com/thoas/go-funk"
)

type timers struct {
	id          uuid.UUID
	timers      []iosomething.TimedCommand
	lastTimerID uint16
}

func NewTimers(id uuid.UUID) iosomething.Handler {
	return &timers{
		id,
		make([]iosomething.TimedCommand, 0),
		1,
	}
}

func (h *timers) timeTicker(remote *iosomething.Remote) *time.Ticker {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		lastMinute := 61
		for tick := range ticker.C {
			if tick.Minute() == lastMinute {
				continue
			}
			lastMinute = tick.Minute()
			for _, timer := range h.timers {
				timerTime := time.Time(timer.Time)
				if timerTime.Hour() == tick.Hour() && timerTime.Minute() == tick.Minute() {
					command, err := json.Marshal(iosomething.DigitalCommand{
						Command: timer.Command,
						Pin:     timer.Pin,
					})
					if err != nil {
						logrus.Fatal("cannot marshal a digitalcommand: ", err)
					}
					remote.PubSub.Pub(
						iosomething.NewMessage(iosomething.DigitalCommandType, h.id, h.id, command),
						iosomething.DigitalCommandType.String())
				}
			}
		}
	}()
	return ticker
}

func (h *timers) addTimer(timer iosomething.TimedCommand) {
	timer.TimerID = h.lastTimerID
	h.lastTimerID++
	h.timers = append(h.timers, timer)
}

func (h *timers) deleteTimer(timer iosomething.TimedCommand) {
	h.timers = funk.Filter(h.timers, func(t iosomething.TimedCommand) bool {
		if t.TimerID == timer.TimerID {
			return false
		}
		return true
	}).([]iosomething.TimedCommand)
}

func (h *timers) editTimer(timer iosomething.TimedCommand) {
	logrus.Warning("Editing not yet implemented")
}

func (h *timers) HandlerRoutine(remote *iosomething.Remote) {
	ticker := h.timeTicker(remote)
	defer ticker.Stop()

	incoming := remote.PubSub.Sub(iosomething.TimedCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*iosomething.Message)
		command := iosomething.TimedCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Warn("Failed decoding timer: ", err)
			continue
		}

		if command.TimerID == 0 {
			h.addTimer(command)
		} else if command.Command == iosomething.DELETE_TIMER {
			h.deleteTimer(command)
		} else {
			h.editTimer(command)
		}
	}
}

func (h *timers) Status() interface{} {
	return h.timers
}

func (h *timers) String() string {
	return "timers"
}
