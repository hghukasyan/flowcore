package flowcore

import (
	"time"
)

// StepFunc is the unit of work executed by the engine.
type StepFunc func(ctx *Context) error

// BackoffKind selects how long to wait between retries.
type BackoffKind int

const (
	BackoffNone BackoffKind = iota
	BackoffFixed
	BackoffExponential
)

// Backoff configures delay between retry attempts (after the first failure).
type Backoff struct {
	Kind       BackoffKind
	BaseDelay  time.Duration // fixed delay, or initial delay for exponential
	Multiplier float64       // for exponential; default 2 if zero
}

// Step describes one named step with optional compensation and retry policy.
type Step struct {
	Name         string
	Run          StepFunc
	Compensate   StepFunc
	DependsOn    []string
	MaxAttempts  int // total attempts including the first run; 0 or 1 means no retries
	Backoff      Backoff
	Timeout      time.Duration
}

func (s *Step) attempts() int {
	if s.MaxAttempts <= 0 {
		return 1
	}
	return s.MaxAttempts
}

func (s *Step) backoffMultiplier() float64 {
	if s.Backoff.Multiplier <= 0 {
		return 2
	}
	return s.Backoff.Multiplier
}
