package timer

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
	timers           []types.TimedCommand
	lastTimerID      uint16
	lastTickedMinute int
	status           *chik.StatusHolder
}

// New creates a new handler for timers
func New() chik.Handler {
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
		-1,
		chik.NewStatusHolder("timers"),
	}

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

func (*timers) Dependencies() []string {
	return []string{"status"}
}

func (*timers) Topics() []types.CommandType {
	return []types.CommandType{types.TimerCommandType}
}

func (h *timers) Setup(controller *chik.Controller) chik.Timer {
	h.status.Set(h.timers, controller)
	return chik.NewTimer(30*time.Second, true)
}

func (h *timers) HandleMessage(message *chik.Message, controller *chik.Controller) {
	var command types.TimedCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logrus.Warn("Failed decoding timer: ", err)
		return
	}

	if funk.Contains(command.Action, types.SET) {
		if command.Time.IsZero() {
			logrus.Warning("Cannot add/edit a timer with a null time")
			return
		}
		if command.TimerID == 0 {
			h.addTimer(command)
		} else {
			h.editTimer(command)
		}
	} else if funk.Contains(command.Action, types.RESET) {
		h.deleteTimer(command)
	}

	h.status.Set(h.timers, controller)
}

func (h *timers) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	if tick.Minute() == h.lastTickedMinute {
		return
	}
	h.lastTickedMinute = tick.Minute()
	for _, timer := range h.timers {
		if timer.Time.Hour() == tick.Hour() && timer.Time.Minute() == tick.Minute() {
			controller.Pub(timer.Command, chik.LoopbackID)
		}
	}
}

func (h *timers) Teardown() {}

func (h *timers) String() string {
	return "timers"
}
