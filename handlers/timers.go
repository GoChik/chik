package handlers

import (
	"encoding/json"
	"math"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

const timerStoragePath = "storage.timers"

type timers struct {
	timers      []types.TimedCommand
	lastTimerID uint16
	status      *chik.StatusHolder
}

func NewTimers() chik.Handler {
	var savedTimers []types.TimedCommand
	err := config.GetStruct(timerStoragePath, &savedTimers)
	if err != nil {
		savedTimers := make([]types.TimedCommand, 0)
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
					remote.Pub(timer.Command, chik.LoopbackID)
				}
			}
		}
	}()
	return ticker
}

func (h *timers) addTimer(timer types.TimedCommand) {
	h.lastTimerID = (h.lastTimerID + 1) % math.MaxUint16
	timer.TimerID = h.lastTimerID
	h.timers = append(h.timers, timer)
	config.Set(timerStoragePath, h.timers)
	config.Sync()
}

func (h *timers) deleteTimer(timer types.TimedCommand) {
	h.timers = funk.Filter(h.timers, func(t types.TimedCommand) bool {
		if t.TimerID == timer.TimerID {
			return false
		}
		return true
	}).([]types.TimedCommand)
	config.Set(timerStoragePath, h.timers)
	config.Sync()
}

func (h *timers) editTimer(timer types.TimedCommand) {
	h.timers = funk.Filter(h.timers, func(t types.TimedCommand) bool {
		if t.TimerID == timer.TimerID {
			return false
		}
		return true
	}).([]types.TimedCommand)
	h.addTimer(timer)
}

func (h *timers) Run(remote *chik.Controller) {
	ticker := h.timeTicker(remote)
	defer ticker.Stop()

	h.status.Set(h.timers, remote)

	incoming := remote.Sub(types.TimerCommandType.String())
	for rawMessage := range incoming {
		message := rawMessage.(*chik.Message)
		var command types.TimedCommand
		err := json.Unmarshal(message.Command().Data, &command)
		if err != nil {
			logrus.Warn("Failed decoding timer: ", err)
			continue
		}

		if funk.Contains(command.Action, types.SET) {
			if command.Time.IsZero() {
				logrus.Warning("Cannot add/edit a timer with a null time")
				continue
			}
			if command.TimerID == 0 {
				h.addTimer(command)
			} else {
				h.editTimer(command)
			}
		} else if funk.Contains(command.Action, types.RESET) {
			h.deleteTimer(command)
		}

		h.status.Set(h.timers, remote)
	}
}

func (h *timers) String() string {
	return "timers"
}
