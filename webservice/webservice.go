package main

import (
	"encoding/base64"
	"fmt"
	"iosomething"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)

// CONFFILE configuration file name
const CONFFILE = "client.json"

type qrHandler struct{}

func (h qrHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := iosomething.GetConfPath(CONFFILE)
	if path == "" {
		fmt.Fprintf(w, "Cannot find config file")
		return
	}

	conf := iosomething.ClientConfiguration{}
	err := iosomething.ParseConf(path, &conf)
	if err != nil {
		fmt.Fprintf(w, "Error parsing configuration")
	}

	code, err := qrcode.Encode(conf.Identity, qrcode.Medium, 256)

	if err != nil {
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(code)

	fmt.Fprintf(w,
		"<h1>Appliance identity code</h1>"+
			"<img src=\"data:image/png;base64,%s\"/>", base64Image)
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", qrHandler{})

	server := http.Server{Addr: "0.0.0.0:8000", Handler: mux}
	server.ListenAndServe()
}
