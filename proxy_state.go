package main

import (
	"sync/atomic"
)

type ProxyState uint32

const (
	NEW      ProxyState = 1
	STARTING ProxyState = 2
	RUNNING  ProxyState = 3
	STOPPED  ProxyState = 4
)

func (s ProxyState) String() string {
	if s < NEW || s > STOPPED {
		return "UNKNOWN"
	}
	strings := [...]string{
		"NEW",
		"STARTING",
		"RUNNING",
		"STOPPED",
	}
	return strings[s-1]
}

func AtomicSwap(addr *ProxyState, new ProxyState) (prev ProxyState) {
	return (ProxyState)(atomic.SwapUint32((*uint32)(addr), (uint32)(new)))
}

func AtomicCompareAndSwap(addr *ProxyState, old, new ProxyState) bool {
	return atomic.CompareAndSwapUint32((*uint32)(addr), (uint32)(old), (uint32)(new))
}
