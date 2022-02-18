package datetime

import (
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/gochik/sunrisesunset"
	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("handler", "time").Logger()

const configKey = "time"

type timeConfig struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type data struct {
	Year    int                  `json:"year"`
	Month   int                  `json:"month"`
	Day     int                  `json:"day"`
	Weekday int                  `json:"weekday"`
	Time    types.TimeIndication `json:"time"`
	Sunrise types.TimeIndication `json:"sunrise"`
	Sunset  types.TimeIndication `json:"sunset"`
}

type datetime struct {
	chik.BaseHandler
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
		logger.Warn().Msgf("Cannot get actions form config file: %v", err)
		config.Set(configKey, conf)
	}

	return &datetime{
		data:   data{},
		conf:   conf,
		status: chik.NewStatusHolder("datetime"),
	}
}

func (h *datetime) String() string {
	return "datetime"
}

func (h *datetime) Dependencies() []string {
	return []string{"status"}
}

func (h *datetime) Topics() []types.CommandType {
	return []types.CommandType{} // TODO: adjust timezone?
}

func (h *datetime) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	return chik.Interrupts{Timer: chik.NewTimer(30*time.Second, true)}, nil
}

func (h *datetime) HandleTimerEvent(tick time.Time, controller *chik.Controller) error {
	if time.Unix(int64(h.data.Time), 0).Minute() == tick.Minute() {
		return nil
	}
	if h.data.Day != tick.Day() {
		sunrise, sunset, _ := sunrisesunset.GetSunriseSunset(h.conf.Latitude, h.conf.Longitude, tick)
		h.data.Sunrise = types.TimeIndication(sunrise.Unix())
		h.data.Sunset = types.TimeIndication(sunset.Unix())
	}
	h.data.Year = tick.Year()
	h.data.Month = int(tick.Month())
	h.data.Day = tick.Day()
	h.data.Weekday = int(tick.Weekday())
	h.data.Time = types.TimeIndication(tick.Unix())
	h.status.Set(h.data, controller)
	return nil
}
