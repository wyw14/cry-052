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
	// Eligibility must be established before the target table is touched.
	// A batch that is still pending (or otherwise not in a terminal,
	// rollback-eligible state) must never reach the adapter rollback call,
	// otherwise the target is truncated to the checkpoint while the caller
	// is told the transition is not allowed.
	if !batch.CanRollback() {
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
