package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestRollbackRejectsIneligibleBatchBeforeChangingTarget(t *testing.T) {
	store := memory.New()
	batch := domain.Batch{ID: "pending-rollback", TargetTable: "sample.masked", Status: domain.BatchPending, Version: 1, InitialTargetRows: 3, RollbackCheckpoint: "3", CreatedAt: time.Unix(1, 0)}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	adapter := &rollbackSpyAdapter{count: 3}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 10, time.Now)
	_, err := processor.Rollback(context.Background(), batch.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
	if adapter.rollbackCalls != 0 {
		t.Fatalf("target changed before rollback eligibility was checked: calls=%d", adapter.rollbackCalls)
	}
	stored, _ := store.GetBatch(context.Background(), batch.ID)
	if stored.Status != domain.BatchPending {
		t.Fatalf("ineligible batch status changed to %s during rejected rollback", stored.Status)
	}
}

func TestRollbackLeavesTargetUntouchedForPendingBatch(t *testing.T) {
	store := memory.New()
	batch := domain.Batch{
		ID:                 "still-pending",
		TargetTable:        "sample.masked",
		Status:             domain.BatchPending,
		Version:            1,
		InitialTargetRows:  2,
		RollbackCheckpoint: "2",
		CreatedAt:          time.Unix(2, 0),
	}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	adapter := &rollbackSpyAdapter{count: 2}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 10, time.Now)

	_, err := processor.Rollback(context.Background(), batch.ID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected invalid transition for pending batch, got %v", err)
	}
	if adapter.rollbackCalls != 0 {
		t.Fatalf("pending batch touched target table before being rejected: calls=%d", adapter.rollbackCalls)
	}
	count, err := adapter.Count(context.Background(), batch.TargetTable)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("target rows changed during rejected rollback: got %d want 2", count)
	}
}

type rollbackSpyAdapter struct {
	count         int64
	rollbackCalls int
}

func (*rollbackSpyAdapter) ReadChunk(context.Context, string, string, int) ([]Row, string, error) {
	return nil, "", nil
}
func (*rollbackSpyAdapter) WriteChunk(context.Context, string, string, []Row) (int64, error) {
	return 0, nil
}
func (a *rollbackSpyAdapter) Count(context.Context, string) (int64, error) { return a.count, nil }
func (a *rollbackSpyAdapter) Rollback(context.Context, string, string) error {
	a.rollbackCalls++
	return nil
}
