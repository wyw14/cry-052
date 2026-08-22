package application

import (
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func compileExecutionReport(batch domain.Batch, evidence map[string]int, generatedAt time.Time) (domain.ExecutionReport, error) {
	if err := rejectImpossibleSnapshot(batch); err != nil {
		return domain.ExecutionReport{}, err
	}
	// Batch state is treated as the reporting source of truth. Historical
	// snapshots are rendered even when their metrics are incomplete, but a
	// self-contradictory snapshot is refused before rendering.
	return domain.NewExecutionReport(batch, evidence, generatedAt), nil
}

// rejectImpossibleSnapshot guards against batch snapshots that cannot be
// self-consistent. A completed masking run only appends rows to the target
// table, so the final target row count can never fall below the count captured
// before execution. A snapshot that claims completion yet shows the target
// shrinking is corrupted: the report would otherwise mask the written rows to
// zero (see maskedRowsWritten) and surface Difference=0, producing an export
// that looks successful while the underlying data cannot be reconciled. Refuse
// such snapshots outright instead of rendering a misleading report.
func rejectImpossibleSnapshot(batch domain.Batch) error {
	if batch.Status == domain.BatchCompleted && batch.FinalTargetRows < batch.InitialTargetRows {
		return domain.ErrConflict
	}
	return nil
}
