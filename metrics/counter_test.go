package metrics

import (
	"sync"
	"testing"
)

func TestCounter(t *testing.T) {
	counter := NewCounter()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)

		// Add 500
		go func() {
			for j := 0; j < 100; j++ {
				counter.Add(5)
			}
			wg.Done()
		}()

		// Add 100
		go func() {
			for j := 0; j < 100; j++ {
				counter.Incr()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	var expected uint64 = 100 * (500 + 100)
	actual := counter.Count()
	if actual != expected {
		t.Errorf("Count was incorrect. expected %d != actual %d", expected, actual)
	}
}
