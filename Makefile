all: build run

build:
	go build

run: build
	./tcp-proxy 2>&1

.PHONY: run build
