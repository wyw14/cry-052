package config

import "testing"

func TestLoadRejectsDuplicateNonAdminSessionTokens(t *testing.T) {
	t.Setenv("ADMIN_SESSION_TOKEN", "admin-only")
	t.Setenv("REVIEWER_SESSION_TOKEN", "shared-non-admin")
	t.Setenv("AUDITOR_SESSION_TOKEN", "shared-non-admin")
	t.Setenv("EXECUTOR_SESSION_TOKEN", "executor-only")
	if _, err := Load(); err == nil {
		t.Fatal("expected pairwise token uniqueness validation")
	}
}
