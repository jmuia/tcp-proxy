package main

import (
	"os"
)

var exitSignals = []os.Signal{os.Interrupt}
var statsSignals = []os.Signal{}
