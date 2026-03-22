package flowcore

import "time"

// StepOption configures a step when it is registered on a workflow.
type StepOption func(*Step)

// DependsOn declares steps that must finish successfully before this one runs.
func DependsOn(names ...string) StepOption {
	return func(s *Step) {
		s.DependsOn = append([]string(nil), names...)
	}
}

// Retry sets how many times the step may run in total (first try + retries).
// Example: Retry(3) allows up to 3 attempts.
func Retry(n int) StepOption {
	return func(s *Step) {
		s.MaxAttempts = n
	}
}

// RetryWithBackoff sets total attempts and delay behavior between retries.
func RetryWithBackoff(n int, b Backoff) StepOption {
	return func(s *Step) {
		s.MaxAttempts = n
		s.Backoff = b
	}
}

// WithCompensation runs fn in reverse order if the workflow fails after this step succeeded.
func WithCompensation(fn StepFunc) StepOption {
	return func(s *Step) {
		s.Compensate = fn
	}
}

// WithTimeout bounds a single attempt of the step.
func WithTimeout(d time.Duration) StepOption {
	return func(s *Step) {
		s.Timeout = d
	}
}
