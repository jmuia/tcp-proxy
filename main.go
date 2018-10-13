package main

import (
	"time"
)

func main() {
	proxyConfig := ProxyConfig{
		laddr:    "localhost:8080",
		timeout:  5 * time.Second,
		services: []string{"localhost:8000"},
	}
	tcpProxy := NewTCPProxy(proxyConfig)
	err := tcpProxy.Run()
	if err != nil {
		logger.Error(err)
	}
}
