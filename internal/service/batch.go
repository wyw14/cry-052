package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

type LocalTableAdapter interface {
	ReadChunk(context.Context, string, string, int) ([]Row, string, error)
	WriteChunk(context.Context, string, string, []Row) (int64, error)
	Count(context.Context, string) (int64, error)
	Rollback(context.Context, string, string) error
}

type BatchStateStore interface {
	GetBatch(context.Context, string) (domain.Batch, error)
	SaveBatch(context.Context, domain.Batch, int64) error
}

type BatchProcessor struct {
	adapter   LocalTableAdapter
	store     BatchStateStore
	engine    *PreviewEngine
	chunkSize int
	clock     func() time.Time
	mu        sync.Mutex
	running   map[string]context.CancelFunc
}

func NewBatchProcessor(adapter LocalTableAdapter, store BatchStateStore, engine *PreviewEngine, chunkSize int, clock func() time.Time) *BatchProcessor {
	if chunkSize < 1 {
		chunkSize = 100
	}
	return &BatchProcessor{adapter: adapter, store: store, engine: engine, chunkSize: chunkSize, clock: clock, running: make(map[string]context.CancelFunc)}
}

func (p *BatchProcessor) Run(ctx context.Context, batchID string, plan CompiledPlan) error {
	batch, err := p.store.GetBatch(ctx, batchID)
	if err != nil {
		return err
	}
	previousVersion := batch.Version
	if batch.RollbackCheckpoint == "" {
		initialTargetRows, err := p.adapter.Count(ctx, batch.TargetTable)
		if err != nil {
			return fmt.Errorf("count target before run: %w", err)
		}
		batch.InitialTargetRows = initialTargetRows
		batch.RollbackCheckpoint = fmt.Sprint(initialTargetRows)
	}
	if err := batch.Start(previousVersion, p.clock()); err != nil {
		return err
	}
	if err := p.store.SaveBatch(ctx, batch, previousVersion); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	if err := p.track(batch.ID, cancel); err != nil {
		cancel()
		return err
	}
	defer p.untrack(batch.ID)
	cursor := batch.Progress.LastRowID
	for {
		rows, nextCursor, readErr := p.adapter.ReadChunk(runCtx, batch.SourceTable, cursor, p.chunkSize)
		if readErr != nil {
			if errors.Is(readErr, context.Canceled) {
				return p.cancelled(ctx, batch.ID, readErr)
			}
			return p.fail(ctx, &batch, "SOURCE_READ_FAILED", readErr)
		}
		if len(rows) == 0 {
			break
		}
		transformed, _, _, transformErr := p.engine.Apply(runCtx, plan, rows)
		if transformErr != nil {
			if errors.Is(transformErr, context.Canceled) {
				return p.cancelled(ctx, batch.ID, transformErr)
			}
			return p.fail(ctx, &batch, "TRANSFORM_FAILED", transformErr)
		}
		operationID := batch.ID + ":" + nextCursor
		written, writeErr := p.adapter.WriteChunk(runCtx, batch.TargetTable, operationID, transformed)
		if writeErr != nil {
			if errors.Is(writeErr, context.Canceled) {
				return p.cancelled(ctx, batch.ID, writeErr)
			}
			return p.fail(ctx, &batch, "TARGET_WRITE_FAILED", writeErr)
		}
		beforeProgress := batch.Version
		batch.Progress.Read += int64(len(rows))
		batch.Progress.Written += written
		batch.Progress.Rejected += int64(len(rows)) - written
		batch.Progress.LastRowID = nextCursor
		batch.Version++
		batch.UpdatedAt = p.clock()
		if err := p.store.SaveBatch(ctx, batch, beforeProgress); err != nil {
			return p.failCurrent(ctx, batch.ID, "PROGRESS_SAVE_FAILED", err)
		}
		cursor = nextCursor
		if err := runCtx.Err(); err != nil {
			return p.cancelled(ctx, batch.ID, err)
		}
	}
	finalTargetRows, err := p.adapter.Count(ctx, batch.TargetTable)
	if err != nil {
		return p.fail(ctx, &batch, "TARGET_COUNT_FAILED", err)
	}
	difference := finalTargetRows - (batch.InitialTargetRows + batch.Progress.Written)
	beforeComplete := batch.Version
	if err := batch.Complete(beforeComplete, finalTargetRows, difference, p.clock()); err != nil {
		if difference != 0 {
			return p.fail(ctx, &batch, "RECONCILIATION_FAILED", fmt.Errorf("target row difference is %d", difference))
		}
		return err
	}
	return p.store.SaveBatch(ctx, batch, beforeComplete)
}

func (p *BatchProcessor) failCurrent(ctx context.Context, batchID, code string, cause error) error {
	batch, err := p.store.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("%v; reload failed batch: %w", cause, err)
	}
	return p.fail(ctx, &batch, code, cause)
}

func (p *BatchProcessor) Cancel(ctx context.Context, batchID string) error {
	p.mu.Lock()
	cancel := p.running[batchID]
	p.mu.Unlock()
	if cancel == nil {
		return domain.ErrNotFound
	}
	cancel()
	return nil
}

func (p *BatchProcessor) track(id string, cancel context.CancelFunc) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.running[id]; exists {
		return fmt.Errorf("batch already running: %w", domain.ErrConflict)
	}
	p.running[id] = cancel
	return nil
}

func (p *BatchProcessor) untrack(id string) {
	p.mu.Lock()
	delete(p.running, id)
	p.mu.Unlock()
}

func (p *BatchProcessor) fail(ctx context.Context, batch *domain.Batch, code string, cause error) error {
	previous := batch.Version
	if err := batch.Fail(previous, code, batch.Progress, p.clock()); err != nil {
		return fmt.Errorf("%v; update failed state: %w", cause, err)
	}
	if err := p.store.SaveBatch(ctx, *batch, previous); err != nil {
		return fmt.Errorf("%v; persist failed state: %w", cause, err)
	}
	return cause
}

func (p *BatchProcessor) cancelled(ctx context.Context, batchID string, cause error) error {
	batch, err := p.store.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("%v; reload cancellation state: %w", cause, err)
	}
	previous := batch.Version
	if err := batch.MarkCancelled(previous, p.clock()); err != nil {
		return fmt.Errorf("%v; transition cancellation: %w", cause, err)
	}
	if err := p.store.SaveBatch(ctx, batch, previous); err != nil {
		return fmt.Errorf("%v; persist cancellation: %w", cause, err)
	}
	return cause
}

func (p *BatchProcessor) Rollback(ctx context.Context, batchID string) (domain.Batch, error) {
	batch, err := p.store.GetBatch(ctx, batchID)
	if err != nil {
		return domain.Batch{}, err
	}
	plan, err := newRollbackPlan(batch)
	if err != nil {
		return domain.Batch{}, err
	}
	if err := p.adapter.Rollback(ctx, plan.TargetTable, plan.Checkpoint); err != nil {
		return domain.Batch{}, fmt.Errorf("rollback target: %w", err)
	}
	count, err := p.adapter.Count(ctx, batch.TargetTable)
	if err != nil {
		return domain.Batch{}, fmt.Errorf("verify rollback: %w", err)
	}
	if err := plan.Verify(count); err != nil {
		return domain.Batch{}, err
	}
	previous := batch.Version
	if err := batch.MarkRolledBack(previous, count, p.clock()); err != nil {
		return domain.Batch{}, err
	}
	if err := p.store.SaveBatch(ctx, batch, previous); err != nil {
		return domain.Batch{}, err
	}
	return batch, nil
}
