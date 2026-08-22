package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry052/internal/domain"
)

type Store struct{ pool *pgxpool.Pool }

type DatabaseCapabilities struct {
	TransactionalDDL bool
	AdvisoryLock     bool
	ServerVersion    int
}

// Validate enforces the safety properties the governance migration relies on.
// Transactional DDL is required so the whole migration group commits or rolls
// back atomically; advisory locking is required so concurrent first-time
// initializations do not interleave their DDL. A supported server version is
// required so the migration does not run against an untested engine.
func (c DatabaseCapabilities) Validate() error {
	if !c.TransactionalDDL {
		return fmt.Errorf("database does not support transactional DDL; governance migration must commit atomically")
	}
	if !c.AdvisoryLock {
		return fmt.Errorf("database does not support advisory locking; governance migration cannot serialize concurrent runs")
	}
	if c.ServerVersion != 0 && c.ServerVersion < minGovernanceServerVersion {
		return fmt.Errorf("server version %d below required minimum %d", c.ServerVersion, minGovernanceServerVersion)
	}
	return nil
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	config.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close()                         { s.pool.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func encode(value any) ([]byte, error) { return json.Marshal(value) }
func decode[T any](payload []byte) (T, error) {
	var value T
	err := json.Unmarshal(payload, &value)
	return value, err
}

func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}

func changed(tag pgconn.CommandTag) error {
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
