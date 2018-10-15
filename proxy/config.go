package proxy

import (
	"time"

	"github.com/jmuia/tcp-proxy/health"
)

type Config struct {
	Laddr    string
	Timeout  time.Duration
	Services []string
	Health   health.HealthCheckConfig
}
