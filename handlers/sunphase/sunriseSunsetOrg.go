package sunphase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const timeFormat = "3:04:05 PM"

func fetchSunTime(latitude, longitude float64) (c cache, err error) {
	logrus.Debug("Fetching sunrise/sunset")

	client := http.Client{}
	request, err := http.NewRequest("GET", "http://api.sunrise-sunset.org/json", nil)
	if err != nil {
		logrus.Error("Failed to format suntime request: ", err)
		return
	}
	query := request.URL.Query()
	query.Add("lat", fmt.Sprintf("%f", latitude))
	query.Add("lng", fmt.Sprintf("%f", longitude))
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
		err = fmt.Errorf(status)
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

	sunset, err := time.Parse(time.RFC3339, sunsetRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	var sunriseRaw string
	err = json.Unmarshal(*results["sunrise"], &sunriseRaw)
	if err != nil {
		logrus.Error(err)
		return
	}
	sunrise, err := time.Parse(time.RFC3339, sunriseRaw)
	if err != nil {
		logrus.Error(err)
		return
	}

	c.Sunrise = stime{sunrise.Hour(), sunrise.Minute()}
	c.Sunset = stime{sunset.Hour(), sunset.Minute()}

	return
}
