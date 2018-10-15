package service

import (
	"sync"

	"github.com/jmuia/tcp-proxy/health"
)

type Registry struct {
	lock      sync.RWMutex
	cfg       health.HealthCheckConfig
	services  map[string]*Service
	monitors  map[string]*HealthMonitor
	listeners []UpdateListener
	aggr      chan *Service
}

func NewRegistry(cfg health.HealthCheckConfig) *Registry {
	r := &Registry{
		lock:      sync.RWMutex{},
		cfg:       cfg,
		services:  make(map[string]*Service),
		monitors:  make(map[string]*HealthMonitor),
		listeners: make([]UpdateListener, 0),
		aggr:      make(chan *Service),
	}
	go func() {
		for s := range r.aggr {
			r.lock.RLock()
			for _, l := range r.listeners {
				go func(s *Service, l UpdateListener) {
					l(*s)
				}(s, l)
			}
			r.lock.RUnlock()
		}
	}()
	return r
}

func (r *Registry) Add(addr string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.remove(addr)

	// TODO: perform an initial health check rather than assuming healthy.
	r.services[addr] = &Service{addr, HEALTHY}
	r.monitors[addr] = NewHealthMonitor(r.services[addr], r.cfg)
	r.monitors[addr].AddHealthCheck(health.NewTCPHealthCheck(addr, r.cfg.Timeout))
	r.monitors[addr].RegisterUpdateListener(func(s Service) {
		r.aggr <- &s
	})
	err := r.monitors[addr].Monitor()
	if err != nil {
		r.remove(addr)
	} else {
		go func() { r.aggr <- r.services[addr] }()
	}
	return err
}

func (r *Registry) Remove(addr string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.remove(addr)
}

func (r *Registry) RegisterUpdateListener(listener UpdateListener) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.listeners = append(r.listeners, listener)
}

func (r *Registry) Snapshot() []Service {
	r.lock.RLock()
	defer r.lock.RUnlock()
	services := make([]Service, 0, len(r.services))
	for _, s := range r.services {
		services = append(services, *s)
	}
	return services
}

func (r *Registry) EvictAll() {
	r.lock.Lock()
	defer r.lock.Unlock()
	for addr := range r.services {
		r.remove(addr)
	}
}

func (r *Registry) remove(addr string) {
	s, exists := r.services[addr]
	if exists {
		go func() { r.aggr <- s }()
		delete(r.services, addr)
	}

	m, exists := r.monitors[addr]
	if exists {
		delete(r.monitors, addr)
		m.Stop()
	}
}
