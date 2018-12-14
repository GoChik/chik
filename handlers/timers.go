package handlers

import (
	"encoding/json"
	"math"
	"time"

	"github.com/gochik/chik/config"
	"github.com/gochik/chik"
	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
	"github.com/thoas/go-funk"
)

const timerStoragePath = "storage.timers"

type timers struct {
	timers      []chik.TimedCommand
	lastTimerID uint16
	status      *chik.StatusHolder
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
		chik.NewStatusHolder("timers"),
	}

}

func (h *timers) timeTicker(remote *chik.Controller) *time.Ticker {
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
					remote.PubSub.Pub(chik.NewMessage(uuid.Nil, timer.Command), timer.Command.Type.String())
				}
			}
		}
	}()
	return ticker
}

func (h *timers) addTimer(timer chik.TimedCommand) {
	h.lastTimerID = (h.lastTimerID + 1) % math.MaxUint16
	timer.TimerID = h.lastTimerID
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
	h.timers = funk.Filter(h.timers, func(t chik.TimedCommand) bool {
		if t.TimerID == timer.TimerID {
			return false
		}
		return true
	}).([]chik.TimedCommand)
	h.addTimer(timer)
}

func (h *timers) Run(remote *chik.Controller) {
	ticker := h.timeTicker(remote)
	defer ticker.Stop()

	h.status.SetStatus(h.timers, remote)

	incoming := remote.PubSub.Sub(chik.TimerCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		command := chik.TimedCommand{}
		err := json.Unmarshal(message.Command().Data, &command)
		if err != nil {
			logrus.Warn("Failed decoding timer: ", err)
			continue
		}

		if funk.Contains(command.Action, chik.SET) {
			if command.Time.IsZero() {
				logrus.Warning("Cannot add/edit a timer with a null time")
				continue
			}
			if command.TimerID == 0 {
				h.addTimer(command)
			} else {
				h.editTimer(command)
			}
		} else if funk.Contains(command.Action, chik.RESET) {
			h.deleteTimer(command)
		}

		h.status.SetStatus(h.timers, remote)
	}
}

func (h *timers) String() string {
	return "timers"
}
