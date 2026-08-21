package application

import "testing"

func TestWorkspaceRefreshPublishesOnlyNewestCompleteGeneration(t *testing.T) {
	app := New(Dependencies{})
	reducer := app.WorkspaceState()
	stale := reducer.Begin()
	current := reducer.Begin()
	sections := []WorkspaceSection{WorkspaceSources, WorkspacePolicies, WorkspaceApprovals, WorkspaceBatches, WorkspaceAudit}
	for _, section := range sections {
		if published := reducer.Commit(current, section, []string{"new-" + string(section)}); section != WorkspaceAudit && published {
			t.Errorf("partial generation published at %s", section)
		}
	}
	if reducer.Commit(stale, WorkspaceSources, []string{"stale-source"}) {
		t.Error("stale refresh was allowed to publish")
	}
	snapshot := reducer.Snapshot()
	if snapshot.Revision != current {
		t.Errorf("revision=%d want=%d", snapshot.Revision, current)
	}
	for _, section := range sections {
		values := snapshot.Sections[section]
		if len(values) != 1 || values[0] != "new-"+string(section) {
			t.Errorf("section %s contains mixed refresh data: %#v", section, values)
		}
	}
}
