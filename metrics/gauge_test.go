package metrics

import (
	"testing"
)

func TestUint64Gauge(t *testing.T) {
	var gauge Gauge = NewUint64Gauge(func() uint64 {
		return 1
	})

	val := gauge.Value()
	uint, ok := val.(uint64)
	if !ok {
		t.Error("failed to cast Uint64Gauge value to uint64")
	}

	if uint != 1 {
		t.Errorf("Uint64Gauge expected 1 got %d", uint)
	}
}
