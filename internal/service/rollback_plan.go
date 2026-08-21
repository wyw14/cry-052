package service

import (
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

type rollbackPlan struct {
	TargetTable string
	Checkpoint  string
	Expected    int64
}

func newRollbackPlan(batch domain.Batch) (rollbackPlan, error) {
	if batch.RollbackCheckpoint == "" {
		return rollbackPlan{}, domain.ErrInvalidTransition
	}
	return rollbackPlan{TargetTable: batch.TargetTable, Checkpoint: batch.RollbackCheckpoint, Expected: batch.InitialTargetRows}, nil
}

func (p rollbackPlan) Verify(observed int64) error {
	if observed != p.Expected {
		return fmt.Errorf("rollback row total %d does not match checkpoint %d: %w", observed, p.Expected, domain.ErrConflict)
	}
	return nil
}
