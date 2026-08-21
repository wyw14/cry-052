package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestExecutionReportRejectsImpossibleBatchSnapshot(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	preview := domain.Preview{ID: "preview", SourceTableID: "source", TargetTableID: "target", StrategyCounters: map[string]int{"mask": 3}}
	if err := store.CreatePreview(ctx, preview); err != nil {
		t.Fatal(err)
	}
	batch := domain.Batch{ID: "batch", PreviewID: preview.ID, IdempotencyKey: "report-key", Status: domain.BatchCompleted, Version: 4, Progress: domain.BatchProgress{Read: 3, Written: 3}, InitialTargetRows: 5, FinalTargetRows: 2, ReconciliationDifference: 0}
	if err := store.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	app := New(Dependencies{Store: store, Now: func() time.Time { return time.Unix(10, 0) }})
	_, err := app.ExecutionReport(ctx, Actor{ID: "auditor", Role: "auditor", RequestID: "req-report"}, batch.ID)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected inconsistent report snapshot to be rejected, got %v", err)
	}
}
