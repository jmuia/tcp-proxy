package metrics

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

type Registry interface {
	Register(name string, metric Metric)
	LoadOrRegisterCounter(name string, counter Counter) (Counter, error)
	LoadOrRegisterGauge(name string, gauge Gauge) (Gauge, error)
	All() map[string]Metric
	Counters() map[string]Counter
	Gauges() map[string]Gauge
	Clear()
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

func (r *registry) LoadOrRegisterCounter(name string, counter Counter) (Counter, error) {
	m, _ := r.metrics.LoadOrStore(name, counter)
	c, ok := m.(Counter)
	if !ok {
		return nil, errors.New(fmt.Sprintf("metric named %s is not a Counter", name))
	}
	return c, nil
}

func (r *registry) LoadOrRegisterGauge(name string, gauge Gauge) (Gauge, error) {
	m, _ := r.metrics.LoadOrStore(name, gauge)
	g, ok := m.(Gauge)
	if !ok {
		return nil, errors.New(fmt.Sprintf("metric named %s is not a Gauge", name))
	}
	return g, nil
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

func (r *registry) Clear() {
	r.metrics.Range(func(key, value interface{}) bool {
		r.metrics.Delete(key)
		return true
	})
}
