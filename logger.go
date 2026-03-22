package flowcore

// Logger receives lifecycle events for steps. All methods are optional to implement;
// the engine only calls what you provide via a non-nil logger.
type Logger interface {
	StepStarted(workflowID, stepName string)
	StepSucceeded(workflowID, stepName string)
	StepFailed(workflowID, stepName string, err error)
}

// PrintLogger writes simple lines to stdout. It satisfies Logger.
type PrintLogger struct{}

func (PrintLogger) StepStarted(workflowID, stepName string) {
	println("[flowcore] start  workflow=" + workflowID + " step=" + stepName)
}

func (PrintLogger) StepSucceeded(workflowID, stepName string) {
	println("[flowcore] ok     workflow=" + workflowID + " step=" + stepName)
}

func (PrintLogger) StepFailed(workflowID, stepName string, err error) {
	println("[flowcore] fail   workflow=" + workflowID + " step=" + stepName + " err=" + err.Error())
}
