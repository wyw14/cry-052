package postgres

import (
	"strings"
	"testing"
)

func TestDiagnosisRestartCannotReplayPartiallyAppliedMigrationPlan(t *testing.T) {
	for _, step := range governanceSchema {
		statement := strings.ToUpper(step.sql)
		if !strings.Contains(statement, "IF NOT EXISTS") {
			t.Fatalf("migration step %q is not replay-safe", step.name)
		}
	}
}
