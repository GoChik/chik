VERSION = $(shell git describe --always)
GOFLAGS = -ldflags="-X iosomething/handlers.Version=$(VERSION)"

.PHONY: default rpi_client gpio_client test server deploy help

default: help

rpi_client:
	cd ${PWD}/client; \
	GOOS=linux GOARCH=arm GOARM=6 go build -tags raspberrypi $(GOFLAGS)
	mkdir -p bin/client
	mv client/client bin/client/linux-arm

gpio_client:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	cd ${PWD}/client; \
	go build -tags gpio $(GOFLAGS)
	mkdir -p bin/client
	mv client/client bin/client/$(GOOS)-$(GOARCH)

server:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	cd ${PWD}/server; \
	go build $(GOFLAGS)
	mkdir -p bin/server
	mv server/server bin/server/$(GOOS)-$(GOARCH)

test:
	go test -cover ./... | grep -v vendor/

deploy:
	make rpi_client
	GOOS=linux GOARCH=mipsle make gpio_client
	GOOS=linux GOARCH=amd64 make server
	go-selfupdate -o release/client bin/client $(VERSION)
	go-selfupdate -o release/server bin/server $(VERSION)

clean:
	git clean -dfx

help:
	@echo "make [rpi_client gpio_client server clean deploy test]"
