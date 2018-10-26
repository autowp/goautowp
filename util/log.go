package util

import (
	"fmt"
	"log"
)

// Logger wraps log infrastructure
type Logger struct {
	rollbar *Rollbar
}

// NewLogger Constructor
func NewLogger(config RollbarConfig) *Logger {
	return &Logger{
		rollbar: NewRollbar(config),
	}
}

// Fatal error
func (l *Logger) Fatal(err error) {
	l.rollbar.Critical(err)
	log.Fatal(err)
}

// Fatalf error
func (l *Logger) Fatalf(format string, v ...interface{}) {
	err := fmt.Errorf(format, v...)
	l.Fatal(err)
}

// Warning error
func (l *Logger) Warning(err error) {
	l.rollbar.Warning(err)
	log.Print(err)
}

// Warningf error
func (l *Logger) Warningf(format string, v ...interface{}) {
	err := fmt.Errorf(format, v...)
	l.Warning(err)
}
