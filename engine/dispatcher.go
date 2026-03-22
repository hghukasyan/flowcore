package engine

import (
	"github.com/hghukasyan/flowcore"
)

// PlanParallel returns layers of steps that can run at the same time, in order.
// It is a thin wrapper over [flowcore.Workflow.ExecutionLayers] for callers that
// already depend on the engine package.
func PlanParallel(wf *flowcore.Workflow) ([][]*flowcore.Step, error) {
	return wf.ExecutionLayers()
}
