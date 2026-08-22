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
	// Declared order is the dependency order; it must not be re-sorted by name,
	// otherwise a child table can run before the parent it references.
	want := []string{"create parent", "create child", "index child"}
	if !reflect.DeepEqual(plan.Names, want) {
		t.Fatalf("dependency order changed: got %v want %v", plan.Names, want)
	}
	if !reflect.DeepEqual(plan.Statements, []string{steps[0].sql, steps[1].sql, steps[2].sql}) {
		t.Fatalf("statements not aligned with declared order: got %v", plan.Statements)
	}
	if plan.Checksum == "" {
		t.Fatal("migration plan has no deterministic checksum")
	}
	// The checksum must fingerprint positions, names and statements so that two
	// structurally different plans never share one.
	reordered := []migrationStep{steps[1], steps[0], steps[2]}
	other, err := planMigrations(reordered)
	if err != nil {
		t.Fatalf("plan reordered migrations: %v", err)
	}
	if other.Checksum == plan.Checksum {
		t.Fatal("reordered plans share a checksum; checksum ignores step order")
	}
	if !reflect.DeepEqual(other.Names, []string{"create child", "create parent", "index child"}) {
		t.Fatalf("reordered plan did not preserve its own declared order: %v", other.Names)
	}
	// Re-planning the identical ordered plan reproduces the checksum.
	again, err := planMigrations(steps)
	if err != nil {
		t.Fatalf("plan migrations again: %v", err)
	}
	if again.Checksum != plan.Checksum {
		t.Fatalf("checksum is not stable: %q then %q", plan.Checksum, again.Checksum)
	}

	duplicate := append(append([]migrationStep(nil), steps...), migrationStep{name: "create child", sql: "SELECT 1"})
	if _, err := planMigrations(duplicate); err == nil {
		t.Fatal("duplicate migration identity was accepted")
	}
	if err := (DatabaseCapabilities{TransactionalDDL: true, AdvisoryLock: false, ServerVersion: 160000}).Validate(); err == nil {
		t.Fatal("migration safety capabilities accepted without advisory locking")
	}
	if err := (DatabaseCapabilities{TransactionalDDL: false, AdvisoryLock: true, ServerVersion: 160000}).Validate(); err == nil {
		t.Fatal("migration safety capabilities accepted without transactional DDL")
	}
	if err := (DatabaseCapabilities{TransactionalDDL: true, AdvisoryLock: true, ServerVersion: 150000}).Validate(); err == nil {
		t.Fatal("migration safety capabilities accepted for unsupported server version")
	}
	if err := (DatabaseCapabilities{TransactionalDDL: true, AdvisoryLock: true, ServerVersion: 160000}).Validate(); err != nil {
		t.Fatalf("valid capabilities rejected: %v", err)
	}
}

func TestMigrationPlanRejectsIncompletePlans(t *testing.T) {
	if _, err := planMigrations(nil); err == nil {
		t.Fatal("empty migration plan was accepted")
	}
	if _, err := planMigrations([]migrationStep{{name: "", sql: "SELECT 1"}}); err == nil {
		t.Fatal("migration step without a name was accepted")
	}
	if _, err := planMigrations([]migrationStep{{name: "blank statement", sql: ""}}); err == nil {
		t.Fatal("migration step without a statement was accepted")
	}
}

func TestGovernancePlanIsValidAndDependencyOrdered(t *testing.T) {
	// The real governance schema must plan without error and keep parents
	// before children: table_schemas references data_sources, previews
	// references table_schemas and policy_versions, batches references
	// previews, approvals references policy_versions. Indexes follow their
	// tables.
	plan, err := planMigrations(governanceSchema)
	if err != nil {
		t.Fatalf("plan governance migrations: %v", err)
	}
	wantOrder := []string{
		"data sources", "classified tables", "policy versions", "confirmed previews",
		"masking batches", "policy approvals", "single pending approval",
		"audit events", "audit chronology", "batch operations",
	}
	if !reflect.DeepEqual(plan.Names, wantOrder) {
		t.Fatalf("governance plan order: got %v want %v", plan.Names, wantOrder)
	}
	parentBeforeChild(t, plan.Names, "data sources", "classified tables")
	parentBeforeChild(t, plan.Names, "policy versions", "confirmed previews")
	parentBeforeChild(t, plan.Names, "classified tables", "confirmed previews")
	parentBeforeChild(t, plan.Names, "confirmed previews", "masking batches")
	parentBeforeChild(t, plan.Names, "policy versions", "policy approvals")
	parentBeforeChild(t, plan.Names, "policy approvals", "single pending approval")
	parentBeforeChild(t, plan.Names, "audit events", "audit chronology")
	parentBeforeChild(t, plan.Names, "masking batches", "batch operations")
	if plan.Checksum == "" {
		t.Fatal("governance plan has no checksum")
	}
}

func parentBeforeChild(t *testing.T, names []string, parent, child string) {
	t.Helper()
	pi, ci := indexOf(names, parent), indexOf(names, child)
	if pi >= ci {
		t.Fatalf("parent %q must run before child %q: order=%v", parent, child, names)
	}
}

func indexOf(names []string, target string) int {
	for i, n := range names {
		if n == target {
			return i
		}
	}
	return -1
}

