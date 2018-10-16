package loadbalancer

import (
	"net"

	"github.com/jmuia/tcp-proxy/backend"
)

type LoadBalancer interface {
	NextBackend(c net.Conn) *backend.Backend
	UpdateBackend(s *backend.Backend)
}
