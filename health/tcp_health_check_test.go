package health

import (
	"testing"
	"time"

	proxytesting "github.com/jmuia/tcp-proxy/testing"
)

func TestTCPHealthCheckOk(t *testing.T) {
	backend := proxytesting.NewLocalListener(t)
	defer backend.Close()

	hc := NewTCPHealthCheck(backend.Addr().String(), 10*time.Millisecond)
	err := hc.Check()

	if err != nil {
		t.Errorf("TCPHealthCheck failed: %v", err)
	}
}

func TestTCPHealthCheckFail(t *testing.T) {
	backend := proxytesting.NewLocalListener(t)
	backend.Close()

	hc := NewTCPHealthCheck(backend.Addr().String(), 10*time.Millisecond)
	err := hc.Check()

	if err == nil {
		t.Errorf("TCPHealthCheck passed, but it was expected to fail")
	}
}
