package main

type ProxyState = uint32

const (
	NEW      ProxyState = 1
	STARTING ProxyState = 2
	RUNNING  ProxyState = 3
	STOPPED  ProxyState = 4
)

func ProxyStateString(s ProxyState) string {
	if s < NEW || s > STOPPED {
		return "UNKNOWN"
	}
	strings := [...]string{
		"NEW",
		"STARTING",
		"RUNNING",
		"STOPPED",
	}
	return strings[s - 1]
}
