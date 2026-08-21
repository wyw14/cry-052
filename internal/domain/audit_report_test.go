package domain

import (
	"testing"
	"time"
)

func TestAuditEventSnapshotsMetadataAndReportUsesBatchReconciliation(t *testing.T) {
	now := time.Date(2026, time.August, 21, 11, 0, 0, 0, time.UTC)
	metadata := map[string]string{"preview_id": "preview-52"}
	event := NewAuditEvent("audit-52", "request-52", "admin", "batch.run", "batch/52", AuditIntent, metadata, now)
	metadata["preview_id"] = "mutated"
	if event.Metadata["preview_id"] != "preview-52" || event.RequestID != "request-52" || event.Outcome != AuditIntent {
		t.Fatalf("event=%+v", event)
	}

	batch := Batch{
		ID:                 "batch-52",
		Status:             BatchCompleted,
		InitialTargetRows:  3,
		FinalTargetRows:    7,
		RollbackCheckpoint: "3",
		Progress:           BatchProgress{Read: 5, Written: 4, Rejected: 1},
	}
	counters := map[string]int{"mask": 4}
	report := NewExecutionReport(batch, counters, now)
	counters["mask"] = 99
	if !report.Reconciled() || !report.RollbackReady || report.TargetRows != 4 || report.StrategyCounters["mask"] != 4 {
		t.Fatalf("report=%+v", report)
	}
}

func TestExecutionReportRejectsImpossibleReconciliation(t *testing.T) {
	report := NewExecutionReport(Batch{
		Status:            BatchCompleted,
		InitialTargetRows: 10,
		FinalTargetRows:   8,
		Progress:          BatchProgress{Read: 1},
	}, nil, time.Now())
	if report.TargetRows != 0 || report.Reconciled() {
		t.Fatalf("report=%+v", report)
	}
}
