VERSION = $(shell git describe --always)
GOFLAGS = -ldflags="-X chik/handlers.Version=$(VERSION) -s -w"

.PHONY: default dependencies rpi_client gpio_client test server deploy help

default: help

dependencies:
	go get -u github.com/rferrazz/go-selfupdate
	go get -u golang.org/x/tools/cmd/stringer

rpi_client:
	cd ${PWD}/client && \
	GOOS=linux GOARCH=arm GOARM=6 go build -tags raspberrypi $(GOFLAGS)
	mkdir -p bin/client
	mv client/client bin/client/linux-arm

gpio_client:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	cd ${PWD}/client && \
	go build -tags gpio $(GOFLAGS)
	mkdir -p bin/client
	mv client/client bin/client/$(GOOS)-$(GOARCH)

fake_client:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	cd ${PWD}/client && \
	go build -tags fake $(GOFLAGS)
	mkdir -p bin/client
	mv client/client bin/client/$(GOOS)-$(GOARCH)

server:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	cd ${PWD}/server && \
	go build $(GOFLAGS)
	mkdir -p bin/server
	mv server/server bin/server/$(GOOS)-$(GOARCH)

test:
	go test -cover ./...

deploy:
	make rpi_client
	GOOS=linux GOARCH=mipsle make gpio_client
	GOOS=darwin GOARCH=amd64 make fake_client
	GOOS=linux GOARCH=amd64 make fake_client
	GOOS=linux GOARCH=amd64 make server
	GOOS=darwin GOARCH=amd64 make server
	mkdir -p release/{client,server}
	rm -rf release/{client,server}/*
	@JFROG_CLI_OFFER_CONFIG=false jfrog bt dlv --user=rferrazz --key=$(BINTRAY_API_KEY) rferrazz/IO-Something/client/rolling release/
	go-selfupdate -o release/client bin/client $(VERSION)
	cd release && JFROG_CLI_OFFER_CONFIG=false jfrog bt u --user=rferrazz --key=$(BINTRAY_API_KEY) --override=true --flat=false --publish=true client/ rferrazz/IO-Something/client/rolling
	@JFROG_CLI_OFFER_CONFIG=false jfrog bt dlv --user=rferrazz --key=$(BINTRAY_API_KEY) rferrazz/IO-Something/server/rolling release/
	go-selfupdate -o release/server bin/server $(VERSION)
	@cd release && JFROG_CLI_OFFER_CONFIG=false jfrog bt u --user=rferrazz --key=$(BINTRAY_API_KEY) --override=true --flat=false --publish=true server/ rferrazz/IO-Something/server/rolling

clean:
	git clean -dfx

help:
	@echo "make [rpi_client gpio_client server clean deploy test]"
