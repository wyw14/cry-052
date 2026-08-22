package application

import (
	"fmt"
	"strings"

	"github.com/wyw14/cry052/internal/domain"
)

// workflowWiring records the subset of dependencies used by connected workflows.
// Optional platform adapters (notifier, files, callbacks, scheduler) are
// intentionally excluded from readiness checks because no workflow requires
// them to be present.
type workflowWiring struct {
	storeAvailable    bool
	samplesAvailable  bool
	compilerAvailable bool
	previewAvailable  bool
}

func inspectWorkflowWiring(deps Dependencies) workflowWiring {
	return workflowWiring{
		storeAvailable:    deps.Store != nil,
		samplesAvailable:  deps.Samples != nil,
		compilerAvailable: deps.Compiler != nil,
		previewAvailable:  deps.Preview != nil,
	}
}

// requirePreviewCreation rejects a request before it touches the compiler or
// preview execution engine. Both, together with the governance store and the
// local sample data, are mandatory for preview creation. The previous
// implementation skipped the compiler and execution engine, so a deployment
// that left them unwired (for example a streamlined local configuration where
// catalog browsing still works) crashed with a nil-pointer dereference in
// CreatePreview instead of returning a handleable configuration error.
func (w workflowWiring) requirePreviewCreation() error {
	var missing []string
	if !w.storeAvailable {
		missing = append(missing, "governance store")
	}
	if !w.samplesAvailable {
		missing = append(missing, "local sample data")
	}
	if !w.compilerAvailable {
		missing = append(missing, "strategy compiler")
	}
	if !w.previewAvailable {
		missing = append(missing, "preview execution engine")
	}
	if len(missing) == 0 {
		return nil
	}
	return domain.NewValidationError(fmt.Sprintf("preview workflow is not fully configured, missing required component(s): %s", strings.Join(missing, ", ")))
}
