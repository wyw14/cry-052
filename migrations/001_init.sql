CREATE TABLE IF NOT EXISTS data_sources (
  id text PRIMARY KEY,
  name text NOT NULL UNIQUE,
  kind text NOT NULL,
  status text NOT NULL,
  version bigint NOT NULL,
  created_at timestamptz NOT NULL,
  payload jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS table_schemas (
  id text PRIMARY KEY,
  datasource_id text NOT NULL REFERENCES data_sources(id),
  qualified_name text NOT NULL,
  fingerprint text NOT NULL,
  payload jsonb NOT NULL,
  UNIQUE (datasource_id, qualified_name)
);

CREATE TABLE IF NOT EXISTS policy_versions (
  id text PRIMARY KEY,
  group_id text NOT NULL,
  version integer NOT NULL,
  status text NOT NULL,
  revision bigint NOT NULL,
  created_at timestamptz NOT NULL,
  payload jsonb NOT NULL,
  UNIQUE (group_id, version)
);

CREATE TABLE IF NOT EXISTS previews (
  id text PRIMARY KEY,
  source_table_id text NOT NULL REFERENCES table_schemas(id),
  target_table_id text NOT NULL REFERENCES table_schemas(id),
  policy_version_id text NOT NULL REFERENCES policy_versions(id),
  confirmed_at timestamptz,
  payload jsonb NOT NULL,
  CHECK (source_table_id <> target_table_id)
);

CREATE TABLE IF NOT EXISTS batches (
  id text PRIMARY KEY,
  preview_id text NOT NULL REFERENCES previews(id),
  idempotency_key text NOT NULL UNIQUE,
  status text NOT NULL,
  version bigint NOT NULL,
  created_at timestamptz NOT NULL,
  payload jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS approvals (
  id text PRIMARY KEY,
  policy_version_id text NOT NULL REFERENCES policy_versions(id),
  decision text NOT NULL,
  revision bigint NOT NULL,
  payload jsonb NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS approvals_one_pending ON approvals(policy_version_id) WHERE decision = 'pending';

CREATE TABLE IF NOT EXISTS audit_events (
  id text PRIMARY KEY,
  request_id text NOT NULL,
  actor text NOT NULL,
  action text NOT NULL,
  resource text NOT NULL,
  outcome text NOT NULL,
  created_at timestamptz NOT NULL,
  payload jsonb NOT NULL
);
