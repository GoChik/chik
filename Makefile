.PHONY: default rpi_client gpio_client tests server help

default: help

rpi_client:
	cd ${PWD}/client; \
	go build -tags raspberrypi
	mkdir bin && cp -f client/client bin/

gpio_client:
	cd ${PWD}/client; \
	go build -tags gpio
	mkdir bin && cp -f client/client bin/

server:
	cd ${PWD}/server; \
	go build
	mkdir bin && cp -f server/server bin/

tests:
	go test ./tests

clean:
	git clean -dfx

help:
	@echo "make [rpi_client gpio_client server clean]"
