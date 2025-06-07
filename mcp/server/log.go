package server

import (
	"fmt"
	"io"
	"os"
	"time"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarn:
		return "WARN"
	case LogError:
		return "ERROR"
	case LogFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	Out       io.Writer
	ErrOut    io.Writer
	MinLevel  LogLevel
	ShowTime  bool
	component string
}

func NewLogger(component string) *Logger {
	return &Logger{
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
		MinLevel:  LogInfo, // Default to INFO level
		ShowTime:  true,
		component: component,
	}
}

func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if level < l.MinLevel {
		return
	}

	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}

	var logLine string
	if l.ShowTime {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
		logLine = fmt.Sprintf("[%s] [%s] [%s]: %s", timestamp, level.String(), l.component, formattedMsg)
	} else {
		logLine = fmt.Sprintf("[%s] [%s]: %s", level.String(), l.component, formattedMsg)
	}

	writer := l.Out
	if level >= LogError {
		writer = l.ErrOut
	}

	fmt.Fprintln(writer, logLine)

	if level == LogFatal {
		os.Exit(1)
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LogDebug, msg, args...)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LogInfo, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LogWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LogError, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(LogFatal, msg, args...)
}
