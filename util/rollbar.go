package util

import (
	"log"
	"time"

	lib "github.com/rollbar/rollbar-go"
)

// RollbarConfig config
type RollbarConfig struct {
	Token       string `yaml:"token"`
	Environment string `yaml:"environment"`
	Period      string `yaml:"period"`
}

// Rollbar wrapper with debounce
type Rollbar struct {
	lastTime time.Time
	period   time.Duration
}

// NewRollbar Rollbar Constructor
func NewRollbar(config RollbarConfig) *Rollbar {
	lib.SetToken(config.Token)
	lib.SetEnvironment(config.Environment)

	r := &Rollbar{}

	var err error
	r.period, err = time.ParseDuration(config.Period)

	if err != nil {
		log.Fatalf("Failed to parse duration `%s`", config.Period)
	}

	return r
}

// Critical message
func (r *Rollbar) Critical(interfaces ...interface{}) bool {
	return r.Log(lib.CRIT, interfaces...)
}

// Warning message
func (r *Rollbar) Warning(interfaces ...interface{}) bool {
	return r.Log(lib.WARN, interfaces...)
}

// Log reports
func (r *Rollbar) Log(level string, interfaces ...interface{}) bool {
	nextTime := r.lastTime.Add(r.period)
	now := time.Now()
	if nextTime.Before(now) {
		lib.Log(level, interfaces...)
		r.lastTime = now
		return true
	}

	return false
}

// Wait until reports sent
func (r *Rollbar) Wait() {
	lib.Wait()
}
