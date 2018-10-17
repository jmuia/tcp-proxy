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
	defer registry.EvictAll()

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

func TestAddingAndRemoving(t *testing.T) {
	cfg := health.HealthCheckConfig{
		Timeout:            10 * time.Millisecond,
		Interval:           1 * time.Minute,
		UnhealthyThreshold: 1,
		HealthyThreshold:   1,
	}
	registry := NewRegistry(cfg)

	// Add backends.
	registry.Add("localhost:12345")
	registry.Add("localhost:54321")
	registry.Add("localhost:19293")
	snapshot := registry.Snapshot()
	if len(snapshot) != 3 {
		t.Errorf("expected 3 backends in the registry, was %d: %v", len(snapshot), snapshot)
	}
	assertContains(t, snapshot, "localhost:12345")
	assertContains(t, snapshot, "localhost:54321")
	assertContains(t, snapshot, "localhost:19293")

	// Remove backends.
	registry.Remove("localhost:54321")
	snapshot = registry.Snapshot()
	if len(snapshot) != 2 {
		t.Errorf("expected 2 backends in the registry, was %d: %v", len(snapshot), snapshot)
	}
	assertContains(t, snapshot, "localhost:12345")
	assertContains(t, snapshot, "localhost:19293")

	// Evict all backends.
	registry.EvictAll()
	snapshot = registry.Snapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected 0 backends in the registry, was %d: %v", len(snapshot), snapshot)
	}
}

func assertContains(t *testing.T, snapshot []Backend, addr string) {
	for _, backend := range snapshot {
		if backend.Addr() == addr {
			return
		}
	}
	t.Errorf("expected to find backend with addr %s in %v", addr, snapshot)
}
