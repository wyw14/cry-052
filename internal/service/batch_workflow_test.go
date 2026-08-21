package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestBatchProcessorReconcilesAndRollsBackTarget(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	adapter := newWorkflowTableAdapter()
	now := time.Now
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "people", Fingerprint: "fp", Version: 1, Fields: []domain.FieldSchema{{Name: "name", DataType: "text"}}}
	target := source
	target.ID = "target"
	target.Schema = "masked"
	target.Fingerprint = "target-fp"
	adapter.Seed(source.QualifiedName(), []Row{{"name": "甲"}, {"name": "乙"}, {"name": "丙"}})
	adapter.Seed(target.QualifiedName(), []Row{{"name": "existing"}})
	registry, _ := NewRegistry(KeepTransformer{})
	plan, conflicts, err := NewCompiler(registry).Compile(source, target, []domain.FieldMapping{{SourceField: "name", TargetField: "name", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}})
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("compile %v %+v", err, conflicts)
	}
	batch := domain.Batch{ID: "batch", SourceTable: source.QualifiedName(), TargetTable: target.QualifiedName(), Status: domain.BatchPending, Version: 1, IdempotencyKey: "key", InputFingerprint: "fp", CreatedAt: now()}
	if err := store.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 2, now)
	if err := processor.Run(ctx, batch.ID, plan); err != nil {
		t.Fatal(err)
	}
	completed, _ := store.GetBatch(ctx, batch.ID)
	if completed.Status != domain.BatchCompleted || completed.InitialTargetRows != 1 || completed.FinalTargetRows != 4 || completed.ReconciliationDifference != 0 {
		t.Fatalf("completed=%+v", completed)
	}
	rolledBack, err := processor.Rollback(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	count, _ := adapter.Count(ctx, target.QualifiedName())
	if rolledBack.Status != domain.BatchRolledBack || count != 1 {
		t.Fatalf("rollback=%+v count=%d", rolledBack, count)
	}
}

func TestBatchProcessorRecoversFailedSnapshotWithoutDuplicateWrites(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	adapter := newWorkflowTableAdapter()
	adapter.failNextWrite = true
	now := time.Now
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "recover_source", Fingerprint: "recover-fp", Version: 1, Fields: []domain.FieldSchema{{Name: "id", DataType: "text"}}}
	target := source
	target.ID = "target"
	target.Schema = "masked"
	target.Fingerprint = "target-fp"
	adapter.Seed(source.QualifiedName(), []Row{{"id": "1"}, {"id": "2"}})
	adapter.Seed(target.QualifiedName(), nil)
	registry, _ := NewRegistry(KeepTransformer{})
	plan, _, _ := NewCompiler(registry).Compile(source, target, []domain.FieldMapping{{SourceField: "id", TargetField: "id", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}})
	batch := domain.Batch{ID: "recover", SourceTable: source.QualifiedName(), TargetTable: target.QualifiedName(), Status: domain.BatchPending, Version: 1, IdempotencyKey: "recover-key", InputFingerprint: "recover-fp", CreatedAt: now()}
	if err := store.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 1, now)
	if err := processor.Run(ctx, batch.ID, plan); err == nil {
		t.Fatal("expected first run to fail")
	}
	failed, _ := store.GetBatch(ctx, batch.ID)
	if failed.Status != domain.BatchFailed {
		t.Fatalf("failed=%+v", failed)
	}
	previous := failed.Version
	if err := failed.Recover(previous, "recover-fp", now()); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveBatch(ctx, failed, previous); err != nil {
		t.Fatal(err)
	}
	if err := processor.Run(ctx, batch.ID, plan); err != nil {
		t.Fatal(err)
	}
	completed, _ := store.GetBatch(ctx, batch.ID)
	count, _ := adapter.Count(ctx, target.QualifiedName())
	if completed.Status != domain.BatchCompleted || count != 2 {
		t.Fatalf("completed=%+v count=%d", completed, count)
	}
}

func TestBatchProcessorRecoveryDeduplicatesWriteAfterProgressPersistenceFailure(t *testing.T) {
	ctx := context.Background()
	base := memory.New()
	store := &failAfterWriteStore{Store: base}
	adapter := newWorkflowTableAdapter()
	now := time.Now
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "atomic_source", Fingerprint: "atomic-fp", Version: 1, Fields: []domain.FieldSchema{{Name: "id", DataType: "text"}}}
	target := source
	target.ID, target.Schema, target.Name = "target", "masked", "atomic_target"
	adapter.Seed(source.QualifiedName(), []Row{{"id": "1"}, {"id": "2"}})
	adapter.Seed(target.QualifiedName(), nil)
	registry, _ := NewRegistry(KeepTransformer{})
	plan, _, _ := NewCompiler(registry).Compile(source, target, []domain.FieldMapping{{SourceField: "id", TargetField: "id", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}})
	batch := domain.Batch{ID: "atomic", SourceTable: source.QualifiedName(), TargetTable: target.QualifiedName(), Status: domain.BatchPending, Version: 1, IdempotencyKey: "atomic-key", InputFingerprint: source.Fingerprint, CreatedAt: now()}
	if err := base.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 1, now)
	if err := processor.Run(ctx, batch.ID, plan); err == nil {
		t.Fatal("expected progress persistence failure")
	}
	failed, err := base.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	countAfterFailure, _ := adapter.Count(ctx, target.QualifiedName())
	if failed.Status != domain.BatchFailed || failed.Progress.Written != 0 || countAfterFailure != 1 {
		t.Fatalf("failed=%+v count=%d", failed, countAfterFailure)
	}
	previous := failed.Version
	if err := failed.Recover(previous, source.Fingerprint, now()); err != nil {
		t.Fatal(err)
	}
	if err := base.SaveBatch(ctx, failed, previous); err != nil {
		t.Fatal(err)
	}
	if err := processor.Run(ctx, batch.ID, plan); err != nil {
		t.Fatal(err)
	}
	completed, _ := base.GetBatch(ctx, batch.ID)
	count, _ := adapter.Count(ctx, target.QualifiedName())
	if completed.Status != domain.BatchCompleted || completed.Progress.Written != 2 || count != 2 {
		t.Fatalf("completed=%+v count=%d", completed, count)
	}
}

func TestBatchRollbackCanRetryAfterStatePersistenceFailure(t *testing.T) {
	ctx := context.Background()
	base := memory.New()
	store := &failRollbackStateStore{Store: base}
	adapter := newWorkflowTableAdapter()
	adapter.Seed("masked.target", []Row{{"id": "existing"}, {"id": "masked"}})
	batch := domain.Batch{ID: "rollback-retry", SourceTable: "sample.source", TargetTable: "masked.target", Status: domain.BatchCompleted, Version: 4, InitialTargetRows: 1, FinalTargetRows: 2, RollbackCheckpoint: "1", IdempotencyKey: "rollback-retry-key", CreatedAt: time.Now()}
	if err := base.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	processor := NewBatchProcessor(adapter, store, NewPreviewEngine(), 1, time.Now)
	if _, err := processor.Rollback(ctx, batch.ID); err == nil {
		t.Fatal("expected injected state persistence failure")
	}
	count, _ := adapter.Count(ctx, batch.TargetTable)
	stored, _ := base.GetBatch(ctx, batch.ID)
	if count != 1 || stored.Status != domain.BatchCompleted {
		t.Fatalf("count=%d stored=%+v", count, stored)
	}
	rolledBack, err := processor.Rollback(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Status != domain.BatchRolledBack || rolledBack.FinalTargetRows != 1 {
		t.Fatalf("rolledBack=%+v", rolledBack)
	}
}

type failAfterWriteStore struct {
	*memory.Store
	failed bool
}

type failRollbackStateStore struct {
	*memory.Store
	failed bool
}

func (s *failRollbackStateStore) SaveBatch(ctx context.Context, batch domain.Batch, expected int64) error {
	if !s.failed && batch.Status == domain.BatchRolledBack {
		s.failed = true
		return fmt.Errorf("injected rollback state failure")
	}
	return s.Store.SaveBatch(ctx, batch, expected)
}

func (s *failAfterWriteStore) SaveBatch(ctx context.Context, batch domain.Batch, expected int64) error {
	if !s.failed && batch.Status == domain.BatchRunning && batch.Progress.Written > 0 {
		s.failed = true
		return fmt.Errorf("injected progress persistence failure")
	}
	return s.Store.SaveBatch(ctx, batch, expected)
}

type workflowTableAdapter struct {
	mu            sync.Mutex
	tables        map[string][]Row
	operations    map[string]int64
	failNextWrite bool
}

func newWorkflowTableAdapter() *workflowTableAdapter {
	return &workflowTableAdapter{tables: map[string][]Row{}, operations: map[string]int64{}}
}
func (a *workflowTableAdapter) Seed(table string, rows []Row) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tables[table] = copyWorkflowRows(rows)
}
func (a *workflowTableAdapter) ReadChunk(ctx context.Context, table, cursor string, size int) ([]Row, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	start := 0
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "%d", &start); err != nil {
			return nil, "", err
		}
	}
	end := start + size
	if end > len(a.tables[table]) {
		end = len(a.tables[table])
	}
	if start >= end {
		return nil, cursor, nil
	}
	return copyWorkflowRows(a.tables[table][start:end]), fmt.Sprint(end), nil
}
func (a *workflowTableAdapter) WriteChunk(ctx context.Context, table, operationID string, rows []Row) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failNextWrite {
		a.failNextWrite = false
		return 0, fmt.Errorf("injected local write failure")
	}
	key := table + "\x00" + operationID
	if written, exists := a.operations[key]; exists {
		return written, nil
	}
	a.tables[table] = append(a.tables[table], copyWorkflowRows(rows)...)
	written := int64(len(rows))
	a.operations[key] = written
	return written, nil
}
func (a *workflowTableAdapter) Count(ctx context.Context, table string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return int64(len(a.tables[table])), nil
}
func (a *workflowTableAdapter) Rollback(ctx context.Context, table, checkpoint string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	keep := 0
	if _, err := fmt.Sscanf(checkpoint, "%d", &keep); err != nil {
		return err
	}
	a.tables[table] = a.tables[table][:keep]
	return nil
}
func copyWorkflowRows(rows []Row) []Row {
	result := make([]Row, len(rows))
	for i, row := range rows {
		result[i] = Row{}
		for key, value := range row {
			result[i][key] = value
		}
	}
	return result
}
