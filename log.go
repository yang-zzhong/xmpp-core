package xmppcore

import (
	"fmt"
	"io"
	"log"
	"runtime"
)

type LogLevel string

const (
	Info  = LogLevel("INFO")
	Debug = LogLevel("DEBUG")
	Error = LogLevel("ERROR")
	Fatal = LogLevel("FATAL")
)

type Logger interface {
	Printf(level LogLevel, format string, v ...interface{})
}

type DefaultLogger struct {
	underlying *log.Logger
}

func NewDefaultLogger(w io.Writer) *DefaultLogger {
	return &DefaultLogger{log.New(w, "", log.Ltime)}
}

func (logger *DefaultLogger) Printf(level LogLevel, format string, v ...interface{}) {
	logger.underlying.Printf("%s: %s", string(level), fmt.Sprintf(format, v...))
	if level == Error || level == Fatal {
		buf := make([]byte, 1<<16)
		runtime.Stack(buf, false)
		logger.underlying.Printf("******************* STACK ********************* \n %s \n        ******************** END STACK ********************\n", string(buf))
	}
}
