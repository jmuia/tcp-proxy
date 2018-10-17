package metrics

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Register counters.
	emptyCounter := NewCounter()
	counter := NewCounter()
	registry.Register("emptyCounter", emptyCounter)
	registry.Register("counter", counter)

	// Register gauges.
	var gaugeValue uint64 = 42
	variableGauge := NewUint64Gauge(func() uint64 { return gaugeValue })
	constantGauge := NewUint64Gauge(func() uint64 { return 1 })
	registry.Register("variableGauge", variableGauge)
	registry.Register("constantGauge", constantGauge)

	// Get all metrics.
	all := registry.All()
	if len(all) != 4 {
		t.Errorf("expected 4 metrics in registry, got %d: %v", len(all), all)
	}

	// Counters.
	counters := registry.Counters()
	if len(counters) != 2 {
		t.Errorf("expected 2 counters in registry, got %d: %v", len(counters), counters)
	}

	registryEmptyCounter, exists := counters["emptyCounter"]
	if !exists {
		t.Error("emptyCounter missing from registry")
	}
	if registryEmptyCounter.Count() != 0 {
		t.Errorf("emptyCounter expected to be 0, was %d", registryEmptyCounter.Count())
	}

	counter.Add(20)
	registryCounter, exists := counters["counter"]
	if !exists {
		t.Error("counter missing from registry")
	}
	if registryCounter.Count() != 20 {
		t.Errorf("counter expected to be 20, was %d", registryCounter.Count())
	}

	// Gauges.
	gauges := registry.Gauges()
	if len(gauges) != 2 {
		t.Errorf("expected 2 gauges in registry, got %d: %v", len(gauges), gauges)
	}

	registryConstantGauge, exists := gauges["constantGauge"]
	if !exists {
		t.Error("constantGauge missing from registry")
	}
	registryConstantGaugeValue := registryConstantGauge.Value().(uint64)
	if registryConstantGaugeValue != 1 {
		t.Errorf("constantGauge expected to be 1, was %d", registryConstantGaugeValue)
	}

	gaugeValue = 1337
	registryVariableGauge, exists := gauges["variableGauge"]
	if !exists {
		t.Error("variableGauge missing from registry")
	}
	registryVariableGaugeValue := registryVariableGauge.Value().(uint64)
	if registryVariableGaugeValue != 1337 {
		t.Errorf("variableGauge expected to be 1, was %d", registryVariableGaugeValue)
	}
}
