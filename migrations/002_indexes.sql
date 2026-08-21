CREATE INDEX IF NOT EXISTS audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX IF NOT EXISTS batches_status_created_at ON batches(status, created_at DESC);
CREATE INDEX IF NOT EXISTS policies_status_created_at ON policy_versions(status, created_at DESC);
