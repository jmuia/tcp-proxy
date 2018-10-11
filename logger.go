package main

import (
	"log"
)

var logger *Logger

func init() {
	logger = &Logger{}
}

type Logger struct {
}

func (l *Logger) Error(v ...interface{}) {
	log.Println(append([]interface{}{"ERROR"}, v...)...)
}

func (l *Logger) Warn(v ...interface{}) {
	log.Println(append([]interface{}{"WARN"}, v...)...)
}

func (l *Logger) Info(v ...interface{}) {
	log.Println(append([]interface{}{"INFO"}, v...)...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	format = "INFO " + format + "\n"
	log.Printf(format, v...)
}
