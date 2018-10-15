package service

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/pkg/errors"
)

type healthMonitorState = uint32

const (
	new_    healthMonitorState = 1
	running healthMonitorState = 2
	stopped healthMonitorState = 3
)

type HealthMonitor struct {
	lock            sync.RWMutex
	cfg             health.HealthCheckConfig
	service         *Service
	checks          []health.HealthCheck
	stopc           chan struct{}
	unhealthyStreak int
	healthyStreak   int
	listeners       []UpdateListener
	state           healthMonitorState
}

func NewHealthMonitor(service *Service, cfg health.HealthCheckConfig) *HealthMonitor {
	return &HealthMonitor{
		lock:            sync.RWMutex{},
		cfg:             cfg,
		service:         service,
		checks:          make([]health.HealthCheck, 0),
		stopc:           make(chan struct{}, 1),
		unhealthyStreak: 0,
		healthyStreak:   0,
		listeners:       make([]UpdateListener, 0),
		state:           new_,
	}
}

func (hm *HealthMonitor) AddHealthCheck(hc health.HealthCheck) {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	hm.checks = append(hm.checks, hc)
}

func (hm *HealthMonitor) RegisterUpdateListener(listener UpdateListener) {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	hm.listeners = append(hm.listeners, listener)
}

func (hm *HealthMonitor) Monitor() error {
	swapped := atomic.CompareAndSwapUint32(&hm.state, new_, running)
	if !swapped {
		return errors.New("attempted to start health monitor for %s when not in NEW state")
	}

	errc := make(chan error)
	ticker := time.NewTicker(hm.cfg.Interval)

	// Health checks run in an independent goroutine
	// to ensure a consistent interval.
	go func() {
		defer close(errc)
		for range ticker.C {
			hm.lock.RLock()
			for _, check := range hm.checks {
				go func(hc health.HealthCheck) {
					errc <- hc.Check()
				}(check)
			}
			hm.lock.RUnlock()
		}
	}()

	go func() {
		defer ticker.Stop()
		for {
			// Select with empty default to prioritize stopping.
			select {
			case <-hm.stopc:
				return
			default:
			}

			select {
			case <-hm.stopc:
				return
			case err := <-errc:
				hm.applyHealthCheck(err)
			}
		}
	}()
	return nil
}

func (hm *HealthMonitor) Stop() {
	prev := atomic.SwapUint32(&hm.state, stopped)
	if prev != stopped {
		close(hm.stopc)
	}
}

func (hm *HealthMonitor) applyHealthCheck(err error) {
	if err != nil {
		hm.healthyStreak = 0
		hm.unhealthyStreak = min(hm.unhealthyStreak+1, hm.cfg.UnhealthyThreshold)
		if hm.unhealthyStreak >= hm.cfg.UnhealthyThreshold {
			updated := hm.service.SetState(UNHEALTHY)
			if updated {
				hm.updateListeners(hm.service)
			}
		}
	} else {
		hm.unhealthyStreak = 0
		hm.healthyStreak = min(hm.healthyStreak+1, hm.cfg.HealthyThreshold)
		if hm.healthyStreak >= hm.cfg.HealthyThreshold {
			updated := hm.service.SetState(HEALTHY)
			if updated {
				hm.updateListeners(hm.service)
			}
		}
	}
}

func (hm *HealthMonitor) updateListeners(s *Service) {
	hm.lock.RLock()
	for _, l := range hm.listeners {
		go func(l UpdateListener) { l(*s) }(l)
	}
	hm.lock.RUnlock()
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
