GOFLAGS = -ldflags="-s -w"

.PHONY: default dependencies test help

default: help

dependencies:
	go get -u golang.org/x/tools/cmd/stringer

test:
	go test -cover ./...

clean:
	git clean -dfx

help:
	@echo "make [dependencies test clean]"
