 SHELL=/usr/bin/env bash

 all: build
.PHONY: all

unexport GOFLAGS

ldflags=-X=github.com/yhio/retrieve-server/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

build: retrieve-server retrieve-http
.PHONY: build

retrieve-server:
	rm -f retrieve-server
	go build $(GOFLAGS) -o retrieve-server ./cmd/retrieve-server
.PHONY: retrieve-server

retrieve-http:
	rm -f retrieve-http
	go build $(GOFLAGS) -o retrieve-http ./cmd/retrieve-http
.PHONY: retrieve-http

clean:
	rm -f retrieve-server
	rm -f retrieve-http
	go clean
.PHONY: clean