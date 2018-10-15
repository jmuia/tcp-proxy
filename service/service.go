package service

import (
	"sync/atomic"
)

type Service struct {
	addr        string
	state       State
	activeConns uint64
}

func (s *Service) Addr() string {
	return s.addr
}

func (s *Service) State() State {
	return (State)(atomic.LoadUint32((*uint32)(&s.state)))
}

func (s *Service) SetState(state State) (updated bool) {
	prev := (State)(atomic.SwapUint32((*uint32)(&s.state), (uint32)(state)))
	return prev != state
}

func (s *Service) IncrActiveConns() uint64 {
	return atomic.AddUint64(&s.activeConns, 1)
}

func (s *Service) DecrActiveConns() uint64 {
	return atomic.AddUint64(&s.activeConns, ^uint64(0))
}

func (s *Service) ActiveConns() uint64 {
	return atomic.LoadUint64(&s.activeConns)
}
