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

build: $(shell find src -name \*.go) $(shell find src -name \*.twig)  ## Build the binary executable
	GOOS=linux GOARCH=amd64 \
		go build \
		-ldflags "$(LD_FLAGS_VERSION)" \
		-o "build/$(NAME)" \
		src/main.go

release: build  ## Build and compress
	upx --ultra-brute "build/$(NAME)"

clean:  ## Remove built files
	rm --force "build/$(NAME)"

start:  ## Start the web app, using Docker
	docker compose --progress plain up --build --force-recreate --detach
