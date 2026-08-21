INSERT INTO data_sources(id, name, kind, status, version, created_at, payload)
VALUES ('demo-source', '本地演示库', 'local_sample', 'ready', 1, now(), '{"id":"demo-source","name":"本地演示库","kind":"local_sample","connection_reference":"secret/demo","status":"ready","version":1,"created_at":"2026-08-21T00:00:00Z","updated_at":"2026-08-21T00:00:00Z"}')
ON CONFLICT (id) DO NOTHING;
