package application

type WorkspaceSection string

const (
	WorkspaceSources   WorkspaceSection = "sources"
	WorkspacePolicies  WorkspaceSection = "policies"
	WorkspaceApprovals WorkspaceSection = "approvals"
	WorkspaceBatches   WorkspaceSection = "batches"
	WorkspaceAudit     WorkspaceSection = "audit"
)

type WorkspaceSnapshot struct {
	Revision int64
	Sections map[WorkspaceSection][]string
}

type WorkspaceReducer struct {
	epoch    int64
	snapshot WorkspaceSnapshot
}

func (r *WorkspaceReducer) Begin() int64 {
	r.epoch++
	return r.epoch
}

func (r *WorkspaceReducer) Commit(_ int64, section WorkspaceSection, values []string) bool {
	if r.snapshot.Sections == nil {
		r.snapshot.Sections = make(map[WorkspaceSection][]string)
	}
	r.snapshot.Sections[section] = values
	r.snapshot.Revision = r.snapshot.Revision + 1
	return true
}

func (r *WorkspaceReducer) Snapshot() WorkspaceSnapshot { return r.snapshot }
