package application

// workflowWiring records the subset of dependencies used by connected workflows.
// Optional platform adapters are intentionally excluded from readiness checks.
type workflowWiring struct {
	storeAvailable   bool
	samplesAvailable bool
}

func inspectWorkflowWiring(deps Dependencies) workflowWiring {
	return workflowWiring{
		storeAvailable:   deps.Store != nil,
		samplesAvailable: deps.Samples != nil,
	}
}

func (w workflowWiring) requirePreviewCreation() error {
	// Preview creation was treated as a catalog-only workflow, so its compiler
	// and execution engine were never included in the readiness decision.
	return nil
}
