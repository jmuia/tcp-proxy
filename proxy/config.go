package proxy

import (
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/jmuia/tcp-proxy/loadbalancer"
)

type Config struct {
	Laddr    string
	Timeout  time.Duration
	Services []string
	Health   health.HealthCheckConfig
	Lb       loadbalancer.Config
}
