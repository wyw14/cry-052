package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestExternalCancellationPersistsTerminalBatchState(t *testing.T) {
	store := memory.New()
	adapter := &blockingTableAdapter{started: make(chan struct{})}
	batch := domain.Batch{ID: "external-cancel", SourceTable: "sample.source", TargetTable: "sample.target", Status: domain.BatchPending, Version: 1, IdempotencyKey: "external-key", InputFingerprint: "snapshot", CreatedAt: time.Unix(1, 0)}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 1, time.Now)
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- processor.Run(runCtx, batch.ID, CompiledPlan{}) }()
	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not enter the source read")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation cause, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("processor did not join after cancellation")
	}
	actual, err := store.GetBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Status != domain.BatchCancelled {
		t.Fatalf("external cancellation left batch in %s", actual.Status)
	}
}
