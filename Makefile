BINARY  := gf-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: build run clean fmt vet test dist

## build: compile for current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## run: build and run (pass args via ARGS="TPE NRT -d 2026-05-01")
run: build
	./$(BINARY) $(ARGS)

## fmt: format all Go source files
fmt:
	go fmt ./...

## vet: run go vet
vet:
	go vet ./...

## test: run tests
test:
	go test ./...

## dist: cross-compile for all platforms
dist:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .
	@echo "Built $(VERSION) to dist/"

## clean: remove built binaries
clean:
	rm -f $(BINARY)
	rm -rf dist/

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## //'
