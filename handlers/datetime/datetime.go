package datetime

import (
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
)

type data struct {
	Year   int `json:"year"`
	Month  int `json:"month"`
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type datetime struct {
	data   data
	status *chik.StatusHolder
}

// New creates a new DateTime handler.
// it updates the global status with the current date and time once every minute
// it allows to execute actions based on the current time
func New() chik.Handler {
	return &datetime{
		data:   data{Minute: -1},
		status: chik.NewStatusHolder("time"),
	}
}

func (h *datetime) String() string {
	return "time"
}

func (h *datetime) Dependencies() []string {
	return []string{"status"}
}

func (h *datetime) Topics() []types.CommandType {
	return []types.CommandType{} // TODO: adjust timezone?
}

func (h *datetime) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewTimer(30*time.Second, true)
}

func (h *datetime) HandleMessage(message *chik.Message, controller *chik.Controller) {}

func (h *datetime) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	if h.data.Minute == tick.Minute() {
		return
	}
	h.data.Year = tick.Year()
	h.data.Month = int(tick.Month())
	h.data.Day = tick.Day()
	h.data.Hour = tick.Hour()
	h.data.Minute = tick.Minute()
	h.status.Set(h.data, controller)
}

func (h *datetime) Teardown() {}
