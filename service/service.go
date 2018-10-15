package service

import (
	"sync/atomic"
)

type ServiceState uint32

const (
	HEALTHY   ServiceState = 1
	UNHEALTHY ServiceState = 2
)

func (s ServiceState) String() string {
	strings := [...]string{"HEALTHY", "UNHEALTHY"}
	switch s {
	case HEALTHY, UNHEALTHY:
		return strings[s-1]
	default:
		return "UNKNOWN"
	}
}

type Service struct {
	addr  string
	state ServiceState
}

func (s *Service) Addr() string {
	return s.addr
}

func (s *Service) State() ServiceState {
	return (ServiceState)(atomic.LoadUint32((*uint32)(&s.state)))
}

func (s *Service) SetState(state ServiceState) (updated bool) {
	prev := (ServiceState)(atomic.SwapUint32((*uint32)(&s.state), (uint32)(state)))
	return prev != state
}
