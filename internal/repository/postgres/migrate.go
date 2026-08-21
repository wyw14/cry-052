package postgres

import (
	"context"
	"fmt"
	"sort"
)

type migrationStep struct {
	name string
	sql  string
}

type MigrationPlan struct {
	Names      []string
	Statements []string
	Checksum   string
}

func planMigrations(steps []migrationStep) (MigrationPlan, error) {
	copySteps := append([]migrationStep(nil), steps...)
	sort.Slice(copySteps, func(i, j int) bool { return copySteps[i].name < copySteps[j].name })
	plan := MigrationPlan{}
	for _, step := range copySteps {
		plan.Names = append(plan.Names, step.name)
		plan.Statements = append(plan.Statements, step.sql)
	}
	return plan, nil
}

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

func (s *Store) Migrate(ctx context.Context) error {
	transaction, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin governance migration: %w", err)
	}
	defer func() { _ = transaction.Rollback(context.WithoutCancel(ctx)) }()
	for sequence, step := range governanceSchema {
		if _, err := transaction.Exec(ctx, step.sql); err != nil {
			return fmt.Errorf("migration %02d (%s): %w", sequence+1, step.name, err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit governance migration: %w", err)
	}
	return nil
}
