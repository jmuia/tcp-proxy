package main

import (
	"sync"
)

type ServiceUpdateListener func(service Service)

type ServiceRegistry struct {
	lock      sync.RWMutex
	cfg       HealthCheckConfig
	services  map[string]*Service
	monitors  map[string]*ServiceHealthMonitor
	listeners []ServiceUpdateListener
	aggr      chan *Service
}

func NewServiceRegistry(cfg HealthCheckConfig) *ServiceRegistry {
	sr := &ServiceRegistry{
		lock:      sync.RWMutex{},
		cfg:       cfg,
		services:  make(map[string]*Service),
		monitors:  make(map[string]*ServiceHealthMonitor),
		listeners: make([]ServiceUpdateListener, 0),
		aggr:      make(chan *Service),
	}
	go func() {
		for s := range sr.aggr {
			sr.lock.RLock()
			for _, l := range sr.listeners {
				go func() { l(*s) }()
			}
			sr.lock.RUnlock()
		}
	}()
	return sr
}

func (sr *ServiceRegistry) Add(addr string) error {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	sr.remove(addr)

	// TODO: perform an initial health check rather than assuming healthy.
	sr.services[addr] = &Service{addr, HEALTHY}
	sr.monitors[addr] = NewServiceHealthMonitor(sr.services[addr], sr.cfg)
	sr.monitors[addr].AddHealthCheck(NewTCPHealthCheck(addr, sr.cfg.timeout))
	sr.monitors[addr].RegisterUpdateListener(func(s Service) {
		sr.aggr <- &s
	})
	err := sr.monitors[addr].Monitor()
	if err != nil {
		sr.remove(addr)
	} else {
		go func() { sr.aggr <- sr.services[addr] }()
	}
	return err
}

func (sr *ServiceRegistry) Remove(addr string) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	sr.remove(addr)
}

func (sr *ServiceRegistry) RegisterUpdateListener(listener ServiceUpdateListener) {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	sr.listeners = append(sr.listeners, listener)
}

func (sr *ServiceRegistry) Snapshot() []Service {
	sr.lock.RLock()
	defer sr.lock.RUnlock()
	services := make([]Service, 0, len(sr.services))
	for _, s := range sr.services {
		services = append(services, *s)
	}
	return services
}

func (sr *ServiceRegistry) EvictAll() {
	sr.lock.Lock()
	defer sr.lock.Unlock()
	for addr := range sr.services {
		sr.remove(addr)
	}
}

func (sr *ServiceRegistry) remove(addr string) {
	s, exists := sr.services[addr]
	if exists {
		go func() { sr.aggr <- s }()
		delete(sr.services, addr)
	}

	m, exists := sr.monitors[addr]
	if exists {
		delete(sr.monitors, addr)
		m.Stop()
	}
}
