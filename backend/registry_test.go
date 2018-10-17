package backend

import (
	"testing"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	proxytesting "github.com/jmuia/tcp-proxy/testing"
)

func TestUpdates(t *testing.T) {
	cfg := health.HealthCheckConfig{
		Timeout:            10 * time.Millisecond,
		Interval:           20 * time.Millisecond,
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	}
	registry := NewRegistry(cfg)

	updatec := make(chan Backend)
	registry.RegisterUpdateListener(func(backend *Backend) {
		updatec <- *backend
	})

	backend1 := proxytesting.NewLocalListener(t)
	backend2 := proxytesting.NewLocalListener(t)
	registry.Add(backend1.Addr().String())
	registry.Add(backend2.Addr().String())

	// Receive an update for each backend that's added as HEALTHY.
	update := <-updatec
	updateState := update.State()
	if updateState != HEALTHY {
		t.Errorf("update expected to indicate HEALTHY, was %s", updateState.String())
	}
	update = <-updatec
	updateState = update.State()
	if updateState != HEALTHY {
		t.Errorf("update expected to indicate HEALTHY, was %s", updateState.String())
	}

	// Shutting down backend1 should publish an UNHEALTHY update.
	backend1.Close()
	update = <-updatec
	updateState = update.State()
	if updateState != UNHEALTHY || update.Addr() != backend1.Addr().String() {
		t.Errorf(
			"update expected to indicate backend1 (%s) UNHEALTHY, was %s %s",
			backend1.Addr().String(),
			update.Addr(),
			updateState.String(),
		)
	}

	// Removing backend2 publishes an update.
	// TODO: it should probably provide context about its removal.
	registry.Remove(backend2.Addr().String())
	update = <-updatec
	if updateState != UNHEALTHY || update.Addr() != backend2.Addr().String() {
		t.Errorf(
			"update expected to indicate backend2 (%s) removed as UNHEALTHY, was %s %s",
			backend2.Addr().String(),
			update.Addr(),
			updateState.String(),
		)
	}
}
