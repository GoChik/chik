package plugins

import (
	"encoding/base64"
	"fmt"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)

type qrHandler struct {
	identity string
}

type webService struct {
	handler *qrHandler
	server  http.Server
}

// NewWebServicePlugin creates a web service that listen on port 8000
// and shows a webpage with a Qr code to help configuring the phone application
func NewWebServicePlugin(identity string) Plugin {
	handler := &qrHandler{identity}
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	return &webService{
		handler: handler,
		server:  http.Server{Addr: "0.0.0.0:8000", Handler: mux},
	}
}

func (h *qrHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	code, err := qrcode.Encode(h.identity, qrcode.Medium, 256)

	if err != nil {
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(code)

	fmt.Fprintf(w,
		"<h1>Appliance identity code</h1>"+
			"<img src=\"data:image/png;base64,%s\"/>", base64Image)
}

func (s *webService) Name() string {
	return "webervice"
}

func (s *webService) Start() {
	go s.server.ListenAndServe()
}

func (s *webService) Stop() {
	s.server.Close()
}
