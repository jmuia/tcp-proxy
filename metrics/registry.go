package metrics

import (
	"sync"
)

var GlobalRegistry = NewRegistry()

type Registry interface {
	Register(name string, metric Metric)
	All() map[string]Metric
	Counters() map[string]Counter
	Gauges() map[string]Gauge
}

func NewRegistry() Registry {
	return &registry{sync.Map{}}
}

type registry struct {
	metrics sync.Map
}

func (r *registry) Register(name string, metric Metric) {
	r.metrics.Store(name, metric)
}

func (r *registry) All() map[string]Metric {
	metrics := make(map[string]Metric, 0)
	// Range calls the function sequentially.
	r.metrics.Range(func(key, value interface{}) bool {
		metrics[key.(string)] = value.(Metric)
		return true
	})
	return metrics
}

func (r *registry) Counters() map[string]Counter {
	counters := make(map[string]Counter, 0)
	// Range calls the function sequentially.
	r.metrics.Range(func(key, value interface{}) bool {
		c, ok := value.(Counter)
		if ok {
			counters[key.(string)] = c
		}
		return true
	})
	return counters
}

func (r *registry) Gauges() map[string]Gauge {
	gauges := make(map[string]Gauge, 0)
	// Range calls the function sequentially.
	r.metrics.Range(func(key, value interface{}) bool {
		g, ok := value.(Gauge)
		if ok {
			gauges[key.(string)] = g
		}
		return true
	})
	return gauges
}
