package xmppcore

import (
	"fmt"
	"io"
	"log"
)

type LogLevel int
type LogMode int

const (
	LogDebug   = LogLevel(0)
	LogInfo    = LogLevel(1)
	LogWarning = LogLevel(2)
	LogError   = LogLevel(3)
	LogFatal   = LogLevel(4)
)

type Logger interface {
	Printf(level LogLevel, format string, v ...interface{})
	Writer() io.Writer
}

type XLogger struct {
	underlying *log.Logger
	level      LogLevel
}

func NewLogger(w io.Writer) *XLogger {
	return &XLogger{log.New(w, "", log.Ltime), LogDebug}
}

func (logger *XLogger) SetLogLevel(level LogLevel) {
	logger.level = level
}

func (logger *XLogger) Writer() io.Writer {
	return logger.underlying.Writer()
}

func (logger *XLogger) Printf(level LogLevel, format string, v ...interface{}) {
	if logger.level > level {
		return
	}
	logger.underlying.Printf("%s: %s", logger.levelString(level), fmt.Sprintf(format, v...))
}

func (logger *XLogger) levelString(level LogLevel) string {
	if level > 4 {
		panic("log level overflow")
	}
	return []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"}[level]
}
