package main

import (
	"net"
	"time"
)

type HealthCheck interface {
	Check() error
}

type TCPHealthCheck struct {
	addr    string
	timeout time.Duration
}

func (hc *TCPHealthCheck) Check() error {
	// TODO: TCP half-open connection.
	conn, err := net.DialTimeout("tcp", hc.addr, hc.timeout)
	if err != nil {
		logger.Infof("%s failed tcp health check", hc.addr)
		return err
	}
	conn.Close()
	logger.Infof("%s passed tcp health check", hc.addr)
	return nil
}

func (hc *TCPHealthCheck) Addr() string {
	return hc.addr
}

func NewTCPHealthCheck(addr string, timeout time.Duration) *TCPHealthCheck {
	return &TCPHealthCheck{addr, timeout}
}
