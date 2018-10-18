package health

import (
	"net"
	"time"
)

type TCPHealthCheck struct {
	addr    string
	timeout time.Duration
}

func (hc *TCPHealthCheck) Check() error {
	// TODO: TCP half-open connection.
	conn, err := net.DialTimeout("tcp", hc.addr, hc.timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (hc *TCPHealthCheck) Addr() string {
	return hc.addr
}

func NewTCPHealthCheck(addr string, timeout time.Duration) *TCPHealthCheck {
	return &TCPHealthCheck{addr, timeout}
}
