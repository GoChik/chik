package handlers

import (
	"chik"
	"chik/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gofrs/uuid"
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

func NewSunset() chik.Handler {
	var confError string
	latitude, ok := config.Get("sunset.latitude").(float64)
	if !ok {
		confError = "Cannot read sunset.latitude"
	}
	longitude, ok := config.Get("sunset.longitude").(float64)
	if !ok {
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
		logrus.Fatal(confError)
	}

	handler := suntime{latitude, longitude, cache{}}
	return &handler
}

func requestTimerStatus(remote *chik.Remote) []chik.TimedCommand {
	result := make([]chik.TimedCommand, 0)
	sub := remote.PubSub.SubOnce(chik.StatusReplyCommandType.String())
	statusCommand, err := json.Marshal(chik.SimpleCommand{Command: []chik.CommandType{chik.GET}})
	if err != nil {
		logrus.Error(err)
		return result
	}
	statusRequest := chik.NewMessage(chik.StatusRequestCommandType, uuid.Nil, statusCommand)
	remote.PubSub.Pub(statusRequest, chik.StatusRequestCommandType.String())
	select {
	case statusRaw := <-sub:
		var status map[string]interface{}
		json.Unmarshal(statusRaw.(*chik.Message).Data(), &status)
		err = chik.Decode(status["timers"], &result)
		if err != nil {
			logrus.Error(err)
		}
		return result
	case <-time.After(500 * time.Millisecond):
		logrus.Error("Cannot fetch timer status")
		return result
	}
}

func (h *suntime) updateTimers(remote *chik.Remote) {
	// fetch sun time
	h.fetchSunTime()

	// fetch timers from a status request
	timers := requestTimerStatus(remote)
	logrus.Debug("Timers: ", timers)

	funk.ForEach(timers, func(t chik.TimedCommand) {
		send := false

		if funk.Contains(t.Command, chik.SUNRISE) {
			t.Time = chik.JSONTime{h.cache.sunrise.In(time.Local)}
			send = true
		}

		if funk.Contains(t.Command, chik.SUNSET) {
			t.Time = chik.JSONTime{h.cache.sunset.In(time.Local)}
			send = true
		}

		if send {
			logrus.Debug("Updating timer according to sun time")
			timerCommand, err := json.Marshal(t)
			if err != nil {
				logrus.Error(err)
				return
			}
			timerChangeRequest := chik.NewMessage(chik.TimerCommandType, uuid.Nil, timerCommand)
			remote.PubSub.Pub(timerChangeRequest, chik.TimerCommandType.String())
		}
	})
}

func (h *suntime) worker(remote *chik.Remote) *time.Ticker {
	ticker := time.NewTicker(23 * time.Hour)
	go func() {
		lastDay := time.Now().Day()
		h.updateTimers(remote)
		for tick := range ticker.C {
			if lastDay == tick.Day() {
				continue
			}
			lastDay = tick.Day()
			h.updateTimers(remote)
		}
	}()
	return ticker
}

func (h *suntime) fetchSunTime() {
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
	reply := make(map[string]interface{}, 0)
	err = json.Unmarshal(replyData, &reply)
	if err != nil {
		logrus.Error(err)
		return
	}
	sunetRaw := reply["results"].(map[string]interface{})["sunset"].(string)
	h.cache.sunset, err = time.Parse(time.RFC3339, sunetRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	sunriseRaw := reply["results"].(map[string]interface{})["sunrise"].(string)
	h.cache.sunrise, err = time.Parse(time.RFC3339, sunriseRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debug("Sunrise: ", h.cache.sunrise, " sunset: ", h.cache.sunset)
}

func (h *suntime) addSunTimer(remote *chik.Remote, timer chik.TimedCommand) {
	if funk.Contains(timer.Command, chik.SUNRISE) {
		timer.Time = chik.JSONTime{h.cache.sunrise.In(time.Local)}
	} else if funk.Contains(timer.Command, chik.SUNSET) {
		timer.Time = chik.JSONTime{h.cache.sunset.In(time.Local)}
	} else {
		logrus.Error("Command does not contain sunrise or sunset")
		return
	}

	rawCommand, err := json.Marshal(timer)
	if err != nil {
		logrus.Error(err)
		return
	}
	message := chik.NewMessage(chik.TimerCommandType, uuid.Nil, rawCommand)
	remote.PubSub.Pub(message, chik.TimerCommandType.String())
}

func (h *suntime) editSunTimer(remote *chik.Remote, timer chik.TimedCommand) {
	logrus.Error("Editing sun timers not supported")
}

func (h *suntime) removeSunTimer(remote *chik.Remote, timer chik.TimedCommand) {
	rawCommand, err := json.Marshal(timer)
	if err != nil {
		logrus.Error(err)
		return
	}
	message := chik.NewMessage(chik.TimerCommandType, uuid.Nil, rawCommand)
	remote.PubSub.Pub(message, chik.TimerCommandType.String())
}

func (h *suntime) Run(remote *chik.Remote) {
	worker := h.worker(remote)
	defer worker.Stop()

	sub := remote.PubSub.Sub(chik.SunsetCommandType.String())
	for rawMessage := range sub {
		message := rawMessage.(*chik.Message)
		var command chik.TimedCommand
		err := json.Unmarshal(message.Data(), &command)
		if err != nil {
			logrus.Error("Command parsing failed")
			continue
		}

		if len(command.Command) < 2 {
			logrus.Warning("Unexpected command length, skipping")
			continue
		}

		if funk.Contains(command.Command, chik.SET) || funk.Contains(command.Command, chik.RESET) {
			if command.TimerID == 0 {
				logrus.Debug("Adding a new sun timer: ", command)
				h.addSunTimer(remote, command)
			} else {
				logrus.Debug("Editing sun timer: ", command)
				h.editSunTimer(remote, command)
			}
			continue
		}

		if funk.Contains(command.Command, chik.DELETE) {
			logrus.Debug("Removing sun timer: ", command)
			h.removeSunTimer(remote, command)
			continue
		}

		logrus.Warning("Unexpected sun command received, skipping")
	}
}

func (h *suntime) Status() interface{} {
	return nil
}

func (h *suntime) String() string {
	return "sunset"
}
