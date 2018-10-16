package metrics

import "sync/atomic"

type Counter interface {
	Count() uint64
	Incr()
	Add(delta uint64)
}

// TODO: consider a cell-based counter similar to JDK LongAdder.
type counter struct {
	count uint64
}

func (c *counter) Count() uint64 {
	return atomic.LoadUint64(&c.count)
}

func (c *counter) Incr() uint64 {
	return atomic.AddUint64(&c.count, 1)
}

func (c *counter) Add(delta uint64) uint64 {
	return atomic.AddUint64(&c.count, delta)
}
