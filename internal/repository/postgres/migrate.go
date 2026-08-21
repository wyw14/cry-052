package postgres

import (
	"context"
	"fmt"
)

func (s *Store) Migrate(ctx context.Context) error {
	for sequence, step := range governanceSchema {
		if _, err := s.pool.Exec(ctx, step.sql); err != nil {
			return migrationFailure(sequence, step, err)
		}
	}
	return nil
}

func migrationFailure(sequence int, step migrationStep, cause error) error {
	return fmt.Errorf("migration %02d (%s) stopped after earlier steps committed: %w", sequence+1, step.name, cause)
}
