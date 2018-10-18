package proxy

import (
	"github.com/jmuia/tcp-proxy/backend"
	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/metrics"
)

type proxyStats struct {
	registry metrics.Registry
	requests metrics.Counter
	errors   metrics.Counter
}

func newProxyStats() *proxyStats {
	stats := &proxyStats{
		registry: metrics.NewRegistry(),
		requests: metrics.NewCounter(),
		errors:   metrics.NewCounter(),
	}
	stats.registry.Register("requests", stats.requests)
	stats.registry.Register("errors", stats.errors)
	return stats
}

type ioStats struct {
	tx uint64
	rx uint64
}

type proxyIoStats struct {
	frontend *ioStats
	backend  *ioStats
}

func newProxyIoStats() *proxyIoStats {
	return &proxyIoStats{&ioStats{}, &ioStats{}}
}

func (ps *proxyStats) incrRequests() {
	ps.requests.Incr()
}

func (ps *proxyStats) incrErrors() {
	ps.errors.Incr()
}

func (ps *proxyStats) incrFrontendIoStats(stats *ioStats) {
	ps.incrIoStats("frontend", stats)
}

func (ps *proxyStats) incrBackendIoStats(addr string, stats *ioStats) {
	ps.incrIoStats("backend."+addr, stats)
}

func (ps *proxyStats) backendActiveConnsGauge(backend *backend.Backend) {
	gauge := metrics.NewUint64Gauge(func() uint64 {
		return backend.ActiveConns()
	})
	ps.registry.Register("backend."+backend.Addr()+".active_connections", gauge)
}

// TODO: don't pessimistically create new metrics.
// Most of the time they'll already exist.
func (ps *proxyStats) incrIoStats(name string, stats *ioStats) {
	incr := func(suffix string, bytes *uint64) {
		counter, err := ps.registry.LoadOrRegisterCounter(
			name+suffix,
			metrics.NewCounter(),
		)
		if err != nil {
			logger.Error(err)
		} else {
			counter.Add(*bytes)
		}
	}
	incr(".io.tx", &stats.tx)
	incr(".io.rx", &stats.rx)
}
