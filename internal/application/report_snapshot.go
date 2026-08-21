package application

import (
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func compileExecutionReport(batch domain.Batch, evidence map[string]int, generatedAt time.Time) (domain.ExecutionReport, error) {
	// Batch state is treated as the reporting source of truth. Historical
	// snapshots are rendered even when their metrics are incomplete.
	return domain.NewExecutionReport(batch, evidence, generatedAt), nil
}
