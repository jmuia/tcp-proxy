package main

import (
	"time"
)

type ProxyConfig struct {
	laddr    string
	timeout  time.Duration
	services []string
	health   HealthCheckConfig
}
