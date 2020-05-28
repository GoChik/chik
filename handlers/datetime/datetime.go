package datetime

import (
	"time"

	"github.com/gochik/chik/config"
	"github.com/sirupsen/logrus"

	"github.com/gochik/chik"
	"github.com/gochik/chik/types"
	"github.com/nathan-osman/go-sunrise"
)

const configKey = "localization"

type timeConfig struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type timeIndication struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type data struct {
	Year    int            `json:"year"`
	Month   int            `json:"month"`
	Day     int            `json:"day"`
	Hour    int            `json:"hour"`
	Minute  int            `json:"minute"`
	Sunrise timeIndication `json:"sunrise"`
	Sunset  timeIndication `json:"sunset"`
}

type datetime struct {
	data   data
	conf   timeConfig
	status *chik.StatusHolder
}

// New creates a new DateTime handler.
// it updates the global status with the current date and time once every minute
// it allows to execute actions based on the current time
func New() chik.Handler {
	var conf timeConfig
	err := config.GetStruct(configKey, &conf)
	if err != nil {
		logrus.Warn("Cannot get actions form config file: ", err)
		config.Set(configKey, conf)
	}

	return &datetime{
		data:   data{Minute: -1},
		conf:   conf,
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
	if h.data.Day != tick.Day() {
		sunrise, sunset := sunrise.SunriseSunset(
			h.conf.Latitude, h.conf.Longitude,
			tick.Year(), tick.Month(), tick.Day(),
		)
		localSunrise := sunrise.Local()
		localSunset := sunset.Local()
		h.data.Sunrise.Hour = localSunrise.Hour()
		h.data.Sunrise.Minute = localSunrise.Minute()
		h.data.Sunset.Hour = localSunset.Hour()
		h.data.Sunset.Minute = localSunset.Minute()
	}
	h.data.Year = tick.Year()
	h.data.Month = int(tick.Month())
	h.data.Day = tick.Day()
	h.data.Hour = tick.Hour()
	h.data.Minute = tick.Minute()
	h.status.Set(h.data, controller)
}

func (h *datetime) Teardown() {}
