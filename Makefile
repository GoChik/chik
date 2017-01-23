.PHONY: default rpi_client gpio_client server webservice help

default: help

rpi_client:
	cd ${PWD}/client; \
	go get; \
	go build -tags raspberrypi
	mkdir bin && cp -f client/client bin/

gpio_client:
	cd ${PWD}/client; \
	go get; \
	go get -u "github.com/davecheney/gpio"; \
	go build -tags gpio
	mkdir bin && cp -f client/client bin/

server:
	cd ${PWD}/server; \
	go get; \
	go build
	mkdir bin && cp -f server/server bin/

webservice:
	cd ${PWD}/webservice; \
	go get; \
	go build
	mkdir bin && cp -f webservice/webservice bin/

clean:
	git clean -dfx

help:
	@echo "make [rpi_client gpio_client server webservice clean]"
