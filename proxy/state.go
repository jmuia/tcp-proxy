package proxy

import (
	"sync/atomic"
)

type State uint32

const (
	NEW      State = 1
	STARTING State = 2
	RUNNING  State = 3
	STOPPED  State = 4
)

func (s State) String() string {
	strings := [...]string{
		"NEW",
		"STARTING",
		"RUNNING",
		"STOPPED",
	}
	switch s {
	case NEW, STARTING, RUNNING, STOPPED:
		return strings[s-1]
	default:
		return "UNKNOWN"
	}
}

func AtomicSwap(addr *State, new State) (prev State) {
	return (State)(atomic.SwapUint32((*uint32)(addr), (uint32)(new)))
}

func AtomicCompareAndSwap(addr *State, old, new State) bool {
	return atomic.CompareAndSwapUint32((*uint32)(addr), (uint32)(old), (uint32)(new))
}
