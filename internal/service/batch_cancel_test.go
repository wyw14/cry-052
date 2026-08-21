package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestDiagnosisConcurrentBatchWritersBothCommitStaleVersion(t *testing.T) {
	store := memory.New()
	batch := domain.Batch{ID: "diagnosis-batch", IdempotencyKey: "diagnosis-key", Status: domain.BatchPending, Version: 1, CreatedAt: time.Now()}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)
	var joined sync.WaitGroup
	joined.Add(2)
	var successes atomic.Int32
	var conflicts atomic.Int32
	for worker := 0; worker < 2; worker++ {
		go func() {
			defer joined.Done()
			candidate, err := store.GetBatch(context.Background(), batch.ID)
			if err != nil {
				t.Error(err)
				return
			}
			ready.Done()
			<-start
			candidate.Status = domain.BatchRunning
			candidate.Version = 2
			err = store.SaveBatch(context.Background(), candidate, 1)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, domain.ErrVersionConflict):
				conflicts.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	ready.Wait()
	close(start)
	joined.Wait()
	if successes.Load() != 1 || conflicts.Load() != 1 {
		t.Fatalf("success=%d conflict=%d", successes.Load(), conflicts.Load())
	}
}

func TestBatchProcessorCancellationPersistsCancelledState(t *testing.T) {
	store := memory.New()
	adapter := &blockingTableAdapter{started: make(chan struct{})}
	now := time.Now
	batch := domain.Batch{ID: "cancel-batch", SourceTable: "sample.source", TargetTable: "sample.target", Status: domain.BatchPending, Version: 1, IdempotencyKey: "cancel-key", InputFingerprint: "fp", CreatedAt: now()}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 1, now)
	done := make(chan error, 1)
	go func() { done <- processor.Run(context.Background(), batch.ID, CompiledPlan{}) }()
	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start reading")
	}
	if err := processor.Cancel(context.Background(), batch.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("run error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop run")
	}
	actual, err := store.GetBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Status != domain.BatchCancelled {
		t.Fatalf("status=%s version=%d", actual.Status, actual.Version)
	}
}

type blockingTableAdapter struct{ started chan struct{} }

func (a *blockingTableAdapter) ReadChunk(ctx context.Context, _, _ string, _ int) ([]Row, string, error) {
	select {
	case <-a.started:
	default:
		close(a.started)
	}
	<-ctx.Done()
	return nil, "", ctx.Err()
}
func (*blockingTableAdapter) WriteChunk(context.Context, string, string, []Row) (int64, error) {
	return 0, nil
}
func (*blockingTableAdapter) Count(context.Context, string) (int64, error)   { return 0, nil }
func (*blockingTableAdapter) Rollback(context.Context, string, string) error { return nil }
