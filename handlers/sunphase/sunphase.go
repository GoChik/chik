package sunphase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/status"
	"github.com/gochik/chik/types"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

type sun uint8

const (
	sunRise sun = iota
	sunSet
)

const timeFormat = "3:04:05 PM"

type suntime struct {
	latitude  float64
	longitude float64
	cache     cache
}

type cache struct {
	sunrise time.Time
	sunset  time.Time
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

	handler := suntime{latitude, longitude, cache{}}
	return &handler
}

func requestTimerStatus(controller *chik.Controller) []types.TimedCommand {
	result := make([]types.TimedCommand, 0)
	sub := controller.SubOnce(types.StatusNotificationCommandType.String())
	statusCommand := status.SubscriptionCommand{Command: types.PUSH, Query: "timers"}
	controller.Pub(types.NewCommand(types.StatusSubscriptionCommandType, statusCommand), chik.LoopbackID)
	select {
	case statusRaw := <-sub:
		var status map[string]interface{}
		json.Unmarshal(statusRaw.(*chik.Message).Command().Data, &status)
		err := types.Decode(status["timers"], &result)
		if err != nil {
			logrus.Error(err)
		}
		return result
	case <-time.After(500 * time.Millisecond):
		logrus.Error("Cannot fetch timer status")
		return result
	}
}

func (h *suntime) fetchSunTime() (err error) {
	logrus.Debug("Fetching sunrise/sunset")

	client := http.Client{}
	request, err := http.NewRequest("GET", "http://api.sunrise-sunset.org/json", nil)
	if err != nil {
		logrus.Error("Failed to format suntime request: ", err)
		return
	}
	query := request.URL.Query()
	query.Add("lat", fmt.Sprintf("%f", h.latitude))
	query.Add("lng", fmt.Sprintf("%f", h.longitude))
	query.Add("formatted", "0")
	request.URL.RawQuery = query.Encode()

	logrus.Debug("Request: ", request.URL.String())

	resp, err := client.Do(request)
	if err != nil {
		logrus.Error("Failed to get sunrise/sunset time: ", err)
		return
	}
	defer resp.Body.Close()
	replyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error(err)
		return
	}
	logrus.Debug("Sunrise/set api response: ", string(replyData))
	var reply map[string]*json.RawMessage
	err = json.Unmarshal(replyData, &reply)
	if err != nil {
		logrus.Error(err)
		return
	}
	var status string
	json.Unmarshal(*reply["status"], &status)

	if status != "OK" {
		logrus.Error("Error fetching sunphase data: ", status)
		return
	}

	var results map[string]*json.RawMessage
	json.Unmarshal(*reply["results"], &results)

	var sunsetRaw string
	err = json.Unmarshal(*results["sunset"], &sunsetRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	h.cache.sunset, err = time.Parse(time.RFC3339, sunsetRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	var sunriseRaw string
	err = json.Unmarshal(*results["sunrise"], &sunriseRaw)
	if err != nil {
		logrus.Error(err)
		return err
	}
	h.cache.sunrise, err = time.Parse(time.RFC3339, sunriseRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debug("Sunrise: ", h.cache.sunrise, " sunset: ", h.cache.sunset)
	return
}

func (h *suntime) addSunTimer(controller *chik.Controller, timer types.TimedCommand) {
	if funk.Contains(timer.Action, types.SUNRISE) {
		timer.Time = types.JSONTime{h.cache.sunrise.In(time.Local)}
	} else if funk.Contains(timer.Action, types.SUNSET) {
		timer.Time = types.JSONTime{h.cache.sunset.In(time.Local)}
	} else {
		logrus.Error("Command does not contain sunrise or sunset")
		return
	}

	controller.Pub(types.NewCommand(types.TimerCommandType, timer), chik.LoopbackID)
}

func (h *suntime) editSunTimer(controller *chik.Controller, timer types.TimedCommand) {
	logrus.Error("Editing sun timers not supported")
}

func (h *suntime) removeSunTimer(controller *chik.Controller, timer types.TimedCommand) {
	controller.Pub(types.NewCommand(types.TimerCommandType, timer), chik.LoopbackID)
}

func (h *suntime) Dependencies() []string {
	return []string{"timers"}
}

func (h *suntime) Topics() []types.CommandType {
	return []types.CommandType{types.SunsetCommandType}
}

func (h *suntime) Setup(controller *chik.Controller) chik.Timer {
	return chik.NewTimer(23*time.Hour, true)
}

func (h *suntime) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	if h.cache.sunrise.Day() == tick.Day() {
		return
	}

	// fetch sun time
	err := h.fetchSunTime()
	if err != nil {
		return
	}

	// fetch timers from a status request
	timers := requestTimerStatus(controller)
	logrus.Debug("Timers: ", timers)

	for _, timer := range timers {
		send := false

		if funk.Contains(timer.Action, types.SUNRISE) {
			timer.Time = types.JSONTime{h.cache.sunrise.In(time.Local)}
			send = true
		}

		if funk.Contains(timer.Action, types.SUNSET) {
			timer.Time = types.JSONTime{h.cache.sunset.In(time.Local)}
			send = true
		}

		if send {
			logrus.Debug("Updating timer according to sun time")
			controller.Pub(types.NewCommand(types.TimerCommandType, timer), chik.LoopbackID)
		}
	}
}

func (h *suntime) HandleMessage(message *chik.Message, controller *chik.Controller) {
	var command types.TimedCommand
	err := json.Unmarshal(message.Command().Data, &command)
	if err != nil {
		logrus.Error("Command parsing failed")
		return
	}

	if len(command.Action) < 2 {
		logrus.Warning("Unexpected command length, skipping")
		return
	}

	if funk.Contains(command.Action, types.SET) {
		if command.TimerID == 0 {
			logrus.Debug("Adding a new sun timer: ", command)
			h.addSunTimer(controller, command)
		} else {
			logrus.Debug("Editing sun timer: ", command)
			h.editSunTimer(controller, command)
		}
		return
	}

	if funk.Contains(command.Command, types.RESET) {
		logrus.Debug("Removing sun timer: ", command)
		h.removeSunTimer(controller, command)
		return
	}

	logrus.Warning("Unexpected sun command received, skipping")
}

func (h *suntime) Teardown() {}

func (h *suntime) String() string {
	return "sunphase"
}
