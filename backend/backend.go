package backend

import (
	"sync/atomic"
)

type Backend struct {
	addr        string
	state       State
	activeConns uint64
}

func (b *Backend) Addr() string {
	return b.addr
}

func (b *Backend) State() State {
	return (State)(atomic.LoadUint32((*uint32)(&b.state)))
}

func (b *Backend) SetState(state State) (updated bool) {
	prev := (State)(atomic.SwapUint32((*uint32)(&b.state), (uint32)(state)))
	return prev != state
}

func (b *Backend) IncrActiveConns() uint64 {
	return atomic.AddUint64(&b.activeConns, 1)
}

func (b *Backend) DecrActiveConns() uint64 {
	return atomic.AddUint64(&b.activeConns, ^uint64(0))
}

func (b *Backend) ActiveConns() uint64 {
	return atomic.LoadUint64(&b.activeConns)
}
