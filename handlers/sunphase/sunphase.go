package sunphase

import (
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
)

type stime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type cache struct {
	Sunrise stime `json:"sunrise"`
	Sunset  stime `json:"sunset"`
}
type suntime struct {
	Latitude       float64
	Longitude      float64
	lastUpdatedDay int
	status         *chik.StatusHolder
}

type confError struct {
	Message string
}

func New() chik.Handler {
	var confError string
	var latitude float64
	var longitude float64
	err := config.GetStruct("sunset.latitude", &latitude)
	if err != nil {
		confError = "Cannot read sunset.latitude"
	}
	err = config.GetStruct("sunset.longitude", &longitude)
	if err != nil {
		if confError == "" {
			confError = "Cannot read sunset.longitude"
		} else {
			confError += " and sunset.longitude"
		}
	}

	if confError != "" {
		config.Set("sunset.latitude", 0)
		config.Set("sunset.longitude", 0)
		config.Sync()
		logrus.Warn(confError)
	}

	return &suntime{
		latitude,
		longitude,
		-1,
		chik.NewStatusHolder("sun"),
	}
}

func (h *suntime) Dependencies() []string {
	return []string{"status"}
}

func (h *suntime) Topics() []types.CommandType {
	return []types.CommandType{}
}

func (h *suntime) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewTimer(23*time.Hour, true)
}

func (h *suntime) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	if h.lastUpdatedDay == tick.Day() {
		return
	}

	// fetch sun time
	result, err := fetchSunTime(h.Latitude, h.Longitude)
	if err != nil {
		return
	}

	h.lastUpdatedDay = tick.Day()
	h.status.Set(result, controller)
}

func (h *suntime) HandleMessage(message *chik.Message, controller *chik.Controller) {}

func (h *suntime) Teardown() {}

func (h *suntime) String() string {
	return "sun"
}
