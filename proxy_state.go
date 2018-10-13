package main

type ProxyState = uint32

const (
	READY   ProxyState = 1
	RUNNING ProxyState = 2
	STOPPED ProxyState = 3
)
