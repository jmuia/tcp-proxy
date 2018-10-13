all: build run

build:
	go build

run: build
	./tcp-proxy localhost:8000

.PHONY: run build
