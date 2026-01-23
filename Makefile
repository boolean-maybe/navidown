SHELL := /bin/sh

.PHONY: help test test-all test-verbose test-navidown lint download tidy

help:
	@printf '%s\n' \
		'Targets:' \
		'  help         show this help' \
		'  test         run package tests (./navidown)' \
		'  test-all     run all tests (./...)' \
		'  test-verbose run all tests with -v' \
		'  test-navidown run a specific package test' \
		'  lint         run golangci-lint' \
		'  download     download dependencies' \
		'  tidy         tidy dependencies'

test:
	go test ./navidown

test-all:
	go test ./...

test-verbose:
	go test -v ./...

test-navidown:
	go test ./navidown -run TestViewer_ParsesHeaders

lint:
	golangci-lint run

download:
	go mod download

tidy:
	go mod tidy
