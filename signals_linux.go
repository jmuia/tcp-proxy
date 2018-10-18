package main

import (
	"os"
	"syscall"
)

var exitSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
var statsSignals = []os.Signal{syscall.SIGUSR1}
