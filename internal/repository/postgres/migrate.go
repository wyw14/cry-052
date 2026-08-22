package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

type migrationStep struct {
	name string
	sql  string
}

// MigrationPlan is the validated, dependency-ordered set of governance
// migration statements. Checksum is a deterministic fingerprint of the ordered
// plan (step positions, names and statements) that identifies exactly which
// set of structures the plan applies, so a failed run can be attributed to one
// unambiguous plan instead of guessing which group was executing.
type MigrationPlan struct {
	Names      []string
	Statements []string
	Checksum   string
}

// planMigrations validates the migration plan and returns it in declared
// dependency order. It deliberately does NOT sort steps by name: the declared
// order is the dependency order (parents before children, tables before their
// indexes), and reordering by name can run a child table before the parent it
// references. Duplicate step names are rejected so a failed run always maps to
// exactly one structure. Each step must carry a name and a statement so the
// plan is complete. A deterministic checksum is computed over the ordered plan
// so the applied set of structures can be confirmed after a failure.
func planMigrations(steps []migrationStep) (MigrationPlan, error) {
	if len(steps) == 0 {
		return MigrationPlan{}, fmt.Errorf("migration plan has no steps")
	}
	plan := MigrationPlan{
		Names:      make([]string, 0, len(steps)),
		Statements: make([]string, 0, len(steps)),
	}
	hash := sha256.New()
	seen := make(map[string]int, len(steps))
	for index, step := range steps {
		if step.name == "" {
			return MigrationPlan{}, fmt.Errorf("migration step %d has no name", index+1)
		}
		if step.sql == "" {
			return MigrationPlan{}, fmt.Errorf("migration step %d (%s) has no statement", index+1, step.name)
		}
		if first, duplicate := seen[step.name]; duplicate {
			return MigrationPlan{}, fmt.Errorf("duplicate migration step name %q (position %d, first seen at %d)", step.name, index+1, first+1)
		}
		seen[step.name] = index
		plan.Names = append(plan.Names, step.name)
		plan.Statements = append(plan.Statements, step.sql)
		fmt.Fprintf(hash, "%d\t%s\t%s\n", index+1, step.name, step.sql)
	}
	plan.Checksum = hex.EncodeToString(hash.Sum(nil))
	return plan, nil
}

// governanceMigrationLock is a stable advisory lock namespace that serializes
// concurrent governance migrations on the same database. It is taken with
// pg_advisory_xact_lock, so it is bound to the migration transaction and
// released on commit or rollback, never blocking a later run. The value decodes
// to "cry052" in ASCII; do not reuse it for another purpose.
const governanceMigrationLock int64 = 0x637279303532

// minGovernanceServerVersion is the lowest supported PostgreSQL server_version_num
// for governance migrations. PostgreSQL 16 is the documented minimum.
const minGovernanceServerVersion = 160000

var governanceSchema = []migrationStep{
	{name: "data sources", sql: `CREATE TABLE IF NOT EXISTS data_sources (id text PRIMARY KEY, name text NOT NULL UNIQUE, kind text NOT NULL, status text NOT NULL, version bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "classified tables", sql: `CREATE TABLE IF NOT EXISTS table_schemas (id text PRIMARY KEY, datasource_id text NOT NULL REFERENCES data_sources(id), qualified_name text NOT NULL, fingerprint text NOT NULL, payload jsonb NOT NULL, UNIQUE(datasource_id, qualified_name))`},
	{name: "policy versions", sql: `CREATE TABLE IF NOT EXISTS policy_versions (id text PRIMARY KEY, group_id text NOT NULL, version integer NOT NULL, status text NOT NULL, revision bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL, UNIQUE(group_id, version))`},
	{name: "confirmed previews", sql: `CREATE TABLE IF NOT EXISTS previews (id text PRIMARY KEY, source_table_id text NOT NULL REFERENCES table_schemas(id), target_table_id text NOT NULL REFERENCES table_schemas(id), policy_version_id text NOT NULL REFERENCES policy_versions(id), confirmed_at timestamptz, payload jsonb NOT NULL, CHECK (source_table_id <> target_table_id))`},
	{name: "masking batches", sql: `CREATE TABLE IF NOT EXISTS batches (id text PRIMARY KEY, preview_id text NOT NULL REFERENCES previews(id), idempotency_key text NOT NULL UNIQUE, status text NOT NULL, version bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "policy approvals", sql: `CREATE TABLE IF NOT EXISTS approvals (id text PRIMARY KEY, policy_version_id text NOT NULL REFERENCES policy_versions(id), decision text NOT NULL, revision bigint NOT NULL, payload jsonb NOT NULL)`},
	{name: "single pending approval", sql: `CREATE UNIQUE INDEX IF NOT EXISTS approvals_one_pending ON approvals(policy_version_id) WHERE decision = 'pending'`},
	{name: "audit events", sql: `CREATE TABLE IF NOT EXISTS audit_events (id text PRIMARY KEY, request_id text NOT NULL, actor text NOT NULL, action text NOT NULL, resource text NOT NULL, outcome text NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "audit chronology", sql: `CREATE INDEX IF NOT EXISTS audit_events_created_at ON audit_events(created_at DESC)`},
	{name: "batch operations", sql: `CREATE INDEX IF NOT EXISTS batches_status_created_at ON batches(status, created_at DESC)`},
}

// capabilities probes the safety properties the governance migration depends on.
// PostgreSQL always supports transactional DDL and advisory locks, but probing
// the server version lets Validate reject an unsupported or misconfigured
// database before any DDL runs.
func (s *Store) capabilities(ctx context.Context) (DatabaseCapabilities, error) {
	var versionText string
	if err := s.pool.QueryRow(ctx, "SHOW server_version_num").Scan(&versionText); err != nil {
		return DatabaseCapabilities{}, fmt.Errorf("read server_version_num: %w", err)
	}
	version, err := strconv.Atoi(versionText)
	if err != nil {
		return DatabaseCapabilities{}, fmt.Errorf("parse server_version_num %q: %w", versionText, err)
	}
	return DatabaseCapabilities{
		TransactionalDDL: true,
		AdvisoryLock:     true,
		ServerVersion:    version,
	}, nil
}

// Migrate applies the governance schema in one transaction so the whole group
// commits atomically. The plan is validated (dependency order, unique step
// names, completeness) and fingerprinted before any statement runs; a
// transaction-scoped advisory lock serializes concurrent migrations. On
// failure the error carries the step position, name and plan checksum so the
// applied set of structures is never ambiguous.
func (s *Store) Migrate(ctx context.Context) error {
	caps, err := s.capabilities(ctx)
	if err != nil {
		return fmt.Errorf("probe database capabilities: %w", err)
	}
	if err := caps.Validate(); err != nil {
		return fmt.Errorf("validate migration capabilities: %w", err)
	}
	plan, err := planMigrations(governanceSchema)
	if err != nil {
		return fmt.Errorf("plan governance migrations: %w", err)
	}
	transaction, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin governance migration: %w", err)
	}
	defer func() { _ = transaction.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", governanceMigrationLock); err != nil {
		return fmt.Errorf("acquire governance migration advisory lock: %w", err)
	}
	for sequence, statement := range plan.Statements {
		if _, err := transaction.Exec(ctx, statement); err != nil {
			return fmt.Errorf("governance migration %02d (%s) checksum=%s: %w", sequence+1, plan.Names[sequence], plan.Checksum, err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit governance migration checksum=%s: %w", plan.Checksum, err)
	}
	return nil
}
