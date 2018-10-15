package service

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/pkg/errors"
)

type ServiceHealthMonitorState = uint32

// TODO: rename once I move code around.
const (
	SHMS_NEW     ServiceHealthMonitorState = 1
	SHMS_RUNNING ServiceHealthMonitorState = 2
	SHMS_STOPPED ServiceHealthMonitorState = 3
)

type ServiceHealthMonitor struct {
	lock            sync.RWMutex
	cfg             health.HealthCheckConfig
	service         *Service
	checks          []health.HealthCheck
	stopc           chan struct{}
	unhealthyStreak int
	healthyStreak   int
	listeners       []ServiceUpdateListener
	state           ServiceHealthMonitorState
}

func NewServiceHealthMonitor(service *Service, cfg health.HealthCheckConfig) *ServiceHealthMonitor {
	return &ServiceHealthMonitor{
		lock:            sync.RWMutex{},
		cfg:             cfg,
		service:         service,
		checks:          make([]health.HealthCheck, 0),
		stopc:           make(chan struct{}, 1),
		unhealthyStreak: 0,
		healthyStreak:   0,
		listeners:       make([]ServiceUpdateListener, 0),
		state:           SHMS_NEW,
	}
}

func (shm *ServiceHealthMonitor) AddHealthCheck(hc health.HealthCheck) {
	shm.lock.Lock()
	defer shm.lock.Unlock()
	shm.checks = append(shm.checks, hc)
}

func (shm *ServiceHealthMonitor) RegisterUpdateListener(listener ServiceUpdateListener) {
	shm.lock.Lock()
	defer shm.lock.Unlock()
	shm.listeners = append(shm.listeners, listener)
}

func (shm *ServiceHealthMonitor) Monitor() error {
	swapped := atomic.CompareAndSwapUint32(&shm.state, SHMS_NEW, SHMS_RUNNING)
	if !swapped {
		return errors.New("attempted to start health monitor for %s when not in NEW state")
	}

	errc := make(chan error)
	ticker := time.NewTicker(shm.cfg.Interval)

	// Health checks run in an independent goroutine
	// to ensure a consistent interval.
	go func() {
		defer close(errc)
		for range ticker.C {
			shm.lock.RLock()
			for _, check := range shm.checks {
				go func() { errc <- check.Check() }()
			}
			shm.lock.RUnlock()
		}
	}()

	go func() {
		defer ticker.Stop()
		for {
			// Select with empty default to prioritize stopping.
			select {
			case <-shm.stopc:
				return
			default:
			}

			select {
			case <-shm.stopc:
				return
			case err := <-errc:
				shm.applyHealthCheck(err)
			}
		}
	}()
	return nil
}

func (shm *ServiceHealthMonitor) Stop() {
	prev := atomic.SwapUint32(&shm.state, SHMS_STOPPED)
	if prev != SHMS_STOPPED {
		close(shm.stopc)
	}
}

func (shm *ServiceHealthMonitor) applyHealthCheck(err error) {
	if err != nil {
		shm.healthyStreak = 0
		shm.unhealthyStreak = min(shm.unhealthyStreak+1, shm.cfg.UnhealthyThreshold)
		if shm.unhealthyStreak >= shm.cfg.UnhealthyThreshold {
			updated := shm.service.SetState(UNHEALTHY)
			if updated {
				shm.notifyUpdateListeners(shm.service)
			}
		}
	} else {
		shm.unhealthyStreak = 0
		shm.healthyStreak = min(shm.healthyStreak+1, shm.cfg.HealthyThreshold)
		if shm.healthyStreak >= shm.cfg.HealthyThreshold {
			updated := shm.service.SetState(HEALTHY)
			if updated {
				shm.notifyUpdateListeners(shm.service)
			}
		}
	}
}

func (shm *ServiceHealthMonitor) notifyUpdateListeners(s *Service) {
	shm.lock.RLock()
	for _, l := range shm.listeners {
		go func() { l(*s) }()
	}
	shm.lock.RUnlock()
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
