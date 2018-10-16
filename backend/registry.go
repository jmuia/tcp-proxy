package backend

import (
	"sync"

	"github.com/jmuia/tcp-proxy/health"
)

type Registry struct {
	lock      sync.RWMutex
	cfg       health.HealthCheckConfig
	backends  map[string]*Backend
	monitors  map[string]*HealthMonitor
	listeners []UpdateListener
	aggr      chan *Backend
}

func NewRegistry(cfg health.HealthCheckConfig) *Registry {
	r := &Registry{
		lock:      sync.RWMutex{},
		cfg:       cfg,
		backends:  make(map[string]*Backend),
		monitors:  make(map[string]*HealthMonitor),
		listeners: make([]UpdateListener, 0),
		aggr:      make(chan *Backend),
	}
	go func() {
		for b := range r.aggr {
			r.lock.RLock()
			for _, l := range r.listeners {
				go func(b *Backend, l UpdateListener) {
					l(b)
				}(b, l)
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
	r.backends[addr] = &Backend{addr, HEALTHY, 0}
	r.monitors[addr] = NewHealthMonitor(r.backends[addr], r.cfg)
	r.monitors[addr].AddHealthCheck(health.NewTCPHealthCheck(addr, r.cfg.Timeout))
	r.monitors[addr].RegisterUpdateListener(func(b *Backend) {
		r.aggr <- b
	})
	err := r.monitors[addr].Monitor()
	if err != nil {
		r.remove(addr)
	} else {
		go func() { r.aggr <- r.backends[addr] }()
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

func (r *Registry) Snapshot() []Backend {
	r.lock.RLock()
	defer r.lock.RUnlock()
	backends := make([]Backend, 0, len(r.backends))
	for _, b := range r.backends {
		backends = append(backends, *b)
	}
	return backends
}

func (r *Registry) EvictAll() {
	r.lock.Lock()
	defer r.lock.Unlock()
	for addr := range r.backends {
		r.remove(addr)
	}
}

func (r *Registry) remove(addr string) {
	b, exists := r.backends[addr]
	if exists {
		go func() { r.aggr <- b }()
		delete(r.backends, addr)
	}

	m, exists := r.monitors[addr]
	if exists {
		delete(r.monitors, addr)
		m.Stop()
	}
}
