package logging

import (
	"log"
)

func Error(v ...interface{}) {
	log.Println(append([]interface{}{"ERROR"}, v...)...)
}

func Warn(v ...interface{}) {
	log.Println(append([]interface{}{"WARN"}, v...)...)
}

func Info(v ...interface{}) {
	log.Println(append([]interface{}{"INFO"}, v...)...)
}

func Infof(format string, v ...interface{}) {
	format = "INFO " + format + "\n"
	log.Printf(format, v...)
}
