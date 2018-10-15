package service

import (
	"sync/atomic"
)

type Service struct {
	addr  string
	state State
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
