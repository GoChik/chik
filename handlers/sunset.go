package handlers

import (
	"chik"
	"chik/config"

	"github.com/Sirupsen/logrus"
)

type sun uint8

const (
	sunRise sun = iota
	sunSet
)

type sunset struct {
	latitude  float32
	longitude float32
}

type confError struct {
	Message string
}

func NewSunset() chik.Handler {
	var confError string
	latitude, ok := config.Get("sunset.latitude").(float32)
	if !ok {
		confError = "Cannot read sunset.latitude"
	}
	longitude, ok := config.Get("sunset.longitude").(float32)
	if !ok {
		if confError != "" {
			confError = "Cannot read sunset.longitude"
		} else {
			confError += " and sunset.longitude"
		}
	}

	if confError != "" {
		config.Set("sunset.latitude", float32(0))
		config.Set("sunset.longitude", float32(0))
		config.Sync()
		logrus.Fatal(confError)
	}

	return &sunset{latitude, longitude}
}

func (h *sunset) Run(remote *chik.Remote) {

}

func (h *sunset) Status() interface{} {
	return nil
}

func (h *sunset) String() string {
	return "sunset"
}
