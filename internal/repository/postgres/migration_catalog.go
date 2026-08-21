package postgres

type migrationStep struct {
	name string
	sql  string
}

var governanceSchema = []migrationStep{
	{name: "data sources", sql: `CREATE TABLE data_sources (id text PRIMARY KEY, name text NOT NULL UNIQUE, kind text NOT NULL, status text NOT NULL, version bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "classified tables", sql: `CREATE TABLE table_schemas (id text PRIMARY KEY, datasource_id text NOT NULL REFERENCES data_sources(id), qualified_name text NOT NULL, fingerprint text NOT NULL, payload jsonb NOT NULL, UNIQUE(datasource_id, qualified_name))`},
	{name: "policy versions", sql: `CREATE TABLE policy_versions (id text PRIMARY KEY, group_id text NOT NULL, version integer NOT NULL, status text NOT NULL, revision bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL, UNIQUE(group_id, version))`},
	{name: "confirmed previews", sql: `CREATE TABLE previews (id text PRIMARY KEY, source_table_id text NOT NULL REFERENCES table_schemas(id), target_table_id text NOT NULL REFERENCES table_schemas(id), policy_version_id text NOT NULL REFERENCES policy_versions(id), confirmed_at timestamptz, payload jsonb NOT NULL, CHECK (source_table_id <> target_table_id))`},
	{name: "masking batches", sql: `CREATE TABLE batches (id text PRIMARY KEY, preview_id text NOT NULL REFERENCES previews(id), idempotency_key text NOT NULL UNIQUE, status text NOT NULL, version bigint NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "policy approvals", sql: `CREATE TABLE approvals (id text PRIMARY KEY, policy_version_id text NOT NULL REFERENCES policy_versions(id), decision text NOT NULL, revision bigint NOT NULL, payload jsonb NOT NULL)`},
	{name: "single pending approval", sql: `CREATE UNIQUE INDEX approvals_one_pending ON approvals(policy_version_id) WHERE decision = 'pending'`},
	{name: "audit events", sql: `CREATE TABLE audit_events (id text PRIMARY KEY, request_id text NOT NULL, actor text NOT NULL, action text NOT NULL, resource text NOT NULL, outcome text NOT NULL, created_at timestamptz NOT NULL, payload jsonb NOT NULL)`},
	{name: "audit chronology", sql: `CREATE INDEX audit_events_created_at ON audit_events(created_at DESC)`},
	{name: "batch operations", sql: `CREATE INDEX batches_status_created_at ON batches(status, created_at DESC)`},
}
