package xmppcore

import (
	"fmt"
	"io"
	"log"
	"runtime"
)

type LogLevel int
type LogMode int

const (
	Debug   = LogLevel(0)
	Info    = LogLevel(1)
	Warning = LogLevel(2)
	Error   = LogLevel(3)
	Fatal   = LogLevel(4)

	DebugMode      = LogMode(0)
	ProductionMode = LogMode(1)
)

type Logger interface {
	Printf(level LogLevel, format string, v ...interface{})
	Writer() io.Writer
}

type XLogger struct {
	underlying *log.Logger
	mode       LogMode
}

func NewLogger(w io.Writer) *XLogger {
	return &XLogger{log.New(w, "", log.Ltime), DebugMode}
}

func (logger *XLogger) SetMode(mode LogMode) {
	logger.mode = mode
}

func (logger *XLogger) Writer() io.Writer {
	return logger.underlying.Writer()
}

func (logger *XLogger) Printf(level LogLevel, format string, v ...interface{}) {
	lowest := Debug
	if logger.mode == ProductionMode {
		lowest = Info
	}
	if level < lowest {
		return
	}
	logger.underlying.Printf("%s: %s", logger.levelString(level), fmt.Sprintf(format, v...))
	if level == Error || level == Fatal {
		buf := make([]byte, 1<<16)
		runtime.Stack(buf, false)
		logger.underlying.Printf("******************* STACK ********************* \n %s \n        ******************** END STACK ********************\n", string(buf))
	}
}

func (logger *XLogger) levelString(level LogLevel) string {
	if level > 4 {
		panic("log level overflow")
	}
	return []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"}[level]
}
