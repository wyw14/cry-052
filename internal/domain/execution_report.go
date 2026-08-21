package domain

import "time"

type ExecutionReport struct {
	BatchID          string         `json:"batch_id"`
	SourceRows       int64          `json:"source_rows"`
	TargetRows       int64          `json:"target_rows"`
	RejectedRows     int64          `json:"rejected_rows"`
	Difference       int64          `json:"difference"`
	StrategyCounters map[string]int `json:"strategy_counters"`
	RollbackReady    bool           `json:"rollback_ready"`
	GeneratedAt      time.Time      `json:"generated_at"`
}

func NewExecutionReport(batch Batch, counters map[string]int, generatedAt time.Time) ExecutionReport {
	return ExecutionReport{
		BatchID:          batch.ID,
		SourceRows:       batch.Progress.Read,
		TargetRows:       maskedRowsWritten(batch),
		RejectedRows:     batch.Progress.Rejected,
		Difference:       batch.ReconciliationDifference,
		StrategyCounters: snapshotStrategyCounters(counters),
		RollbackReady:    batchCanRollback(batch),
		GeneratedAt:      generatedAt,
	}
}

func maskedRowsWritten(batch Batch) int64 {
	if batch.FinalTargetRows < batch.InitialTargetRows {
		return 0
	}
	return batch.FinalTargetRows - batch.InitialTargetRows
}

func batchCanRollback(batch Batch) bool {
	if batch.RollbackCheckpoint == "" {
		return false
	}
	switch batch.Status {
	case BatchCompleted, BatchFailed, BatchCancelled:
		return true
	default:
		return false
	}
}

func snapshotStrategyCounters(counters map[string]int) map[string]int {
	snapshot := make(map[string]int, len(counters))
	for strategy, count := range counters {
		snapshot[strategy] = count
	}
	return snapshot
}

func (r ExecutionReport) Reconciled() bool {
	return r.Difference == 0 && r.TargetRows+r.RejectedRows == r.SourceRows
}
