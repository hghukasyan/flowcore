package store

import (
	"context"
	"errors"
)

// StepStatus is the lifecycle state of a single step.
type StepStatus string

const (
	StatusPending StepStatus = "pending"
	StatusRunning StepStatus = "running"
	StatusDone    StepStatus = "done"
	StatusFailed  StepStatus = "failed"
)

// StepState is persisted metadata for one step.
type StepState struct {
	Name       string
	Status     StepStatus
	RetryCount int
}

// WorkflowState is persisted metadata for a workflow run.
type WorkflowState struct {
	ID    string
	Steps map[string]*StepState
}

// Store persists workflow and step status for observability and future backends.
type Store interface {
	// PutWorkflow creates or replaces the workflow shell with pending steps.
	PutWorkflow(ctx context.Context, id string, stepNames []string) error
	// SetStep updates one step's status and retry counter.
	SetStep(ctx context.Context, workflowID, stepName string, status StepStatus, retryCount int) error
	// GetWorkflow returns a snapshot of stored state.
	GetWorkflow(ctx context.Context, id string) (*WorkflowState, error)
}

var ErrNotFound = errors.New("store: workflow not found")
