#! /usr/bin/make

# Name of this app, used as basename for almost everything.
NAME=mpa

# Eg: 2026-05-21 19:17:44
DATE=$(shell date '+%F %T')
# Eg: v1.0.0
VERSION=$(shell git describe --tags --always)

LD_FLAGS_VERSION=-X main/src/version.GitSummary=$(VERSION)

.PHONY: help clean start
.DEFAULT_GOAL := help


help:
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: build-linux-64 build-windows-64 build-mac-64  ## Build all the binary executables

build-linux-64: $(shell find src -name \*)  ## Build the linux binary executable
	GOOS=linux GOARCH=amd64 \
		go build \
		-ldflags "$(LD_FLAGS_VERSION)" \
		-o "build/$(NAME)" \
		src/main.go
	chmod +x "build/$(NAME)"

build-windows-64: $(shell find src -name \*)  ## Build the windows binary executable
	GOOS=windows GOARCH=amd64 \
		go build \
		-ldflags "$(LD_FLAGS_VERSION)" \
		-o "build/$(NAME)_win.exe" \
		src/main.go

build-mac-64: $(shell find src -name \*)  ## Build the apple binary executable
	GOOS=darwin GOARCH=amd64 \
		go build \
		-ldflags "$(LD_FLAGS_VERSION)" \
		-o "build/$(NAME)_mac" \
		src/main.go
	chmod +x "build/$(NAME)_mac"

release: build  ## Build and compress
	upx --ultra-brute --best "build/$(NAME)"
	@# upx: build/mpa_mac: CantPackException: macOS is currently not supported (try --force-macos)
	@#upx --ultra-brute --best "build/$(NAME)_mac"
	upx --ultra-brute --best "build/$(NAME)_win.exe"

clean:  ## Remove built files
	rm --force "build/$(NAME)"
	rm --force "build/$(NAME)_mac"
	rm --force "build/$(NAME)_win.exe"

start:  ## Start the web app, using Docker
	docker compose --progress plain up --build --force-recreate --detach
