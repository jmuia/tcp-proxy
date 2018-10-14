package main

type ProxyState = uint32

const (
	NEW      ProxyState = 1
	STARTING ProxyState = 2
	RUNNING  ProxyState = 3
	STOPPED  ProxyState = 4
)
