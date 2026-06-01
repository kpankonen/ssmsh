SHELL := /bin/bash
PROJECT := github.com/bwhaley/ssmsh
EXECUTABLE := ssmsh
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

.PHONY: build test

VERSION := $(shell echo ${SSMSH_VERSION})
ifeq "$(VERSION)" ""
    VERSION="auto-build"
endif

ifndef $(GOPATH)
   GOPATH=$(shell go env GOPATH)
   export GOPATH
endif

all: test build

GO_LDFLAGS := -X $(shell go list ./$(PACKAGE)).GitCommit=$(GIT_COMMIT) -X main.Version=${VERSION}

test:
	@echo "FORMATTING"
	go fmt ./...
	@echo "VETTING"
	go vet -v ./...
	@echo "LINTING"
	golangci-lint run ./...
	@echo "TESTING"
	go test -v ./...

build:
	go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE) $(PROJECT)
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-linux-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-darwin-amd64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o $(GOPATH)/bin/$(EXECUTABLE)-darwin-arm64

clean:
	rm -f $(GOPATH)/bin/$(EXECUTABLE) $(GOPATH)/bin/$(EXECUTABLE)-*
