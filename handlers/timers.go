package handlers

import (
	"chik"
	"chik/config"
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
	"github.com/thoas/go-funk"
)

const timerStoragePath = "storage.timers"

type timers struct {
	timers      []chik.TimedCommand
	lastTimerID uint16
}

func NewTimers() chik.Handler {
	var savedTimers []chik.TimedCommand
	err := config.GetStruct(timerStoragePath, &savedTimers)
	if err != nil {
		savedTimers := make([]chik.TimedCommand, 0)
		config.Set(timerStoragePath, savedTimers)
		config.Sync()
		logrus.Warning("storage.timers section was invalid. It has been reset: ", err)
	}

	lastID := uint16(1)
	if len(savedTimers) > 0 {
		lastID = savedTimers[len(savedTimers)-1].TimerID
	}

	logrus.Debug("Timers", savedTimers)

	return &timers{
		savedTimers,
		lastID,
	}

}

func (h *timers) timeTicker(remote *chik.Remote) *time.Ticker {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		lastMinute := 61
		for tick := range ticker.C {
			if tick.Minute() == lastMinute {
				continue
			}
			lastMinute = tick.Minute()
			for _, timer := range h.timers {
				if timer.Time.Hour() == tick.Hour() && timer.Time.Minute() == tick.Minute() {
					command, err := json.Marshal(chik.DigitalCommand{
						Command: timer.Command,
						Pin:     timer.Pin,
					})
					if err != nil {
						logrus.Fatal("cannot marshal a digitalcommand: ", err)
					}
					remote.PubSub.Pub(
						chik.NewMessage(chik.DigitalCommandType, uuid.Nil, uuid.Nil, command),
						chik.DigitalCommandType.String())
				}
			}
		}
	}()
	return ticker
}

func (h *timers) addTimer(timer chik.TimedCommand) {
	timer.TimerID = h.lastTimerID
	h.lastTimerID++
	h.timers = append(h.timers, timer)
	config.Set(timerStoragePath, h.timers)
	config.Sync()
}

func (h *timers) deleteTimer(timer chik.TimedCommand) {
	h.timers = funk.Filter(h.timers, func(t chik.TimedCommand) bool {
		if t.TimerID == timer.TimerID {
			return false
		}
		return true
	}).([]chik.TimedCommand)
	config.Set(timerStoragePath, h.timers)
	config.Sync()
}

func (h *timers) editTimer(timer chik.TimedCommand) {
	logrus.Warning("Editing not yet implemented")
}

func (h *timers) Run(remote *chik.Remote) {
	ticker := h.timeTicker(remote)
	defer ticker.Stop()

	incoming := remote.PubSub.Sub(chik.TimerCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		command := chik.TimedCommand{}
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Warn("Failed decoding timer: ", err)
			continue
		}

		if len(command.Command) != 1 {
			logrus.Error("Unexpected composed command")
			continue
		}

		if command.TimerID == 0 {
			h.addTimer(command)
		} else if command.Command[0] == chik.RESET {
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
