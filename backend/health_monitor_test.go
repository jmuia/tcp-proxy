package backend

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/pkg/errors"
)

type fakeHealthCheck func() error

func (hc fakeHealthCheck) Check() error { return hc() }

func TestHealthFlappingAboveThreshold(t *testing.T) {
	backend := &Backend{"localhost:57803", HEALTHY, 0}
	cfg := health.HealthCheckConfig{
		Interval:           5 * time.Millisecond,
		UnhealthyThreshold: 3,
		HealthyThreshold:   3,
	}

	hm := NewHealthMonitor(backend, cfg)

	hcc := make(chan error)
	updatec := make(chan Backend)

	hm.AddHealthCheck(fakeHealthCheck(func() error {
		return <-hcc
	}))
	hm.RegisterUpdateListener(func(backend *Backend) {
		updatec <- *backend
	})

	err := hm.Monitor()
	defer hm.Stop()
	if err != nil {
		t.Fatal(err)
	}

	// Backend starts healthy.
	state := backend.State()
	if state != HEALTHY {
		t.Errorf("backend expected to be HEALTHY, was %s", state.String())
	}

	// Fail health checks to become unhealthy.
	for i := 0; i < cfg.UnhealthyThreshold; i++ {
		hcc <- errors.New("health check failed")
	}

	// We get updated about the state change.
	update := <-updatec
	updateState := update.State()
	if updateState != UNHEALTHY {
		t.Errorf("update expected to indicate UNHEALTHY, was %s", updateState.String())
	}

	// Assert the backend (non-copied) is also unhealthy.
	state = backend.State()
	if state != UNHEALTHY {
		t.Errorf("backend expected to be UNHEALTHY, was %s", state.String())
	}

	// Pass health checks to become healthy.
	for i := 0; i < cfg.UnhealthyThreshold; i++ {
		hcc <- nil
	}

	// We get updated about the state change.
	update = <-updatec
	updateState = update.State()
	if updateState != HEALTHY {
		t.Errorf("update expected to indicate HEALTHY, was %s", updateState.String())
	}

	// Assert the backend (non-copied) is also healthy.
	state = backend.State()
	if state != HEALTHY {
		t.Errorf("backend expected to be HEALTHY, was %s", state.String())
	}
}

func TestHealthFlappingBelowThreshold(t *testing.T) {
	backend := &Backend{"localhost:57803", HEALTHY, 0}
	cfg := health.HealthCheckConfig{
		Interval:           5 * time.Millisecond,
		UnhealthyThreshold: 3,
		HealthyThreshold:   3,
	}

	hm := NewHealthMonitor(backend, cfg)

	hcc := make(chan error)
	var updateCount int32

	hm.AddHealthCheck(fakeHealthCheck(func() error {
		return <-hcc
	}))
	hm.RegisterUpdateListener(func(backend *Backend) {
		atomic.AddInt32(&updateCount, 1)
	})

	err := hm.Monitor()
	defer hm.Stop()
	if err != nil {
		t.Fatal(err)
	}

	// Backend starts healthy.
	state := backend.State()
	if state != HEALTHY {
		t.Errorf("backend expected to be HEALTHY, was %s", state.String())
	}

	// Flap below thresholds.
	for i := 0; i < 5; i++ {
		for j := 0; j < cfg.UnhealthyThreshold-1; j++ {
			hcc <- errors.New("health check failed")
		}
		for j := 0; j < cfg.HealthyThreshold-1; j++ {
			hcc <- nil
		}
	}

	// Backend should still be healthy.
	state = backend.State()
	if state != HEALTHY {
		t.Errorf("backend expected to be HEALTHY, was %s", state.String())
	}

	// There were no state changes, so we didn't receive updates.
	finalCount := atomic.LoadInt32(&updateCount)
	if finalCount != 0 {
		t.Errorf("received %d updates, expected 0", finalCount)
	}
}
