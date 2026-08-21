package postgres

import (
	"reflect"
	"testing"
)

func TestMigrationPlanPreservesDependencyOrderAndRejectsAmbiguity(t *testing.T) {
	steps := []migrationStep{{name: "create parent", sql: "CREATE TABLE parent(id int primary key)"}, {name: "create child", sql: "CREATE TABLE child(parent_id int references parent(id))"}, {name: "index child", sql: "CREATE INDEX child_parent ON child(parent_id)"}}
	plan, err := planMigrations(steps)
	if err != nil {
		t.Fatalf("plan migrations: %v", err)
	}
	want := []string{"create parent", "create child", "index child"}
	if !reflect.DeepEqual(plan.Names, want) {
		t.Fatalf("dependency order changed: got %v want %v", plan.Names, want)
	}
	if plan.Checksum == "" {
		t.Fatal("migration plan has no deterministic checksum")
	}
	duplicate := append(append([]migrationStep(nil), steps...), migrationStep{name: "create child", sql: "SELECT 1"})
	if _, err := planMigrations(duplicate); err == nil {
		t.Fatal("duplicate migration identity was accepted")
	}
	if err := (DatabaseCapabilities{TransactionalDDL: true, AdvisoryLock: false, ServerVersion: 160000}).Validate(); err == nil {
		t.Fatal("migration safety capabilities accepted without advisory locking")
	}
}
