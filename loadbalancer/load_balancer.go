package loadbalancer

import (
	"net"

	"github.com/jmuia/tcp-proxy/service"
)

type LoadBalancer interface {
	NextService(c net.Conn) *service.Service
	UpdateService(s *service.Service)
}
