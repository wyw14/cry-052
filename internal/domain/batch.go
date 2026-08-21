package domain

import (
	"time"
)

type BatchStatus string

const (
	BatchPending    BatchStatus = "pending"
	BatchRunning    BatchStatus = "running"
	BatchCancelling BatchStatus = "cancelling"
	BatchCancelled  BatchStatus = "cancelled"
	BatchFailed     BatchStatus = "failed"
	BatchRecovering BatchStatus = "recovering"
	BatchCompleted  BatchStatus = "completed"
	BatchRolledBack BatchStatus = "rolled_back"
)

type BatchProgress struct {
	Read      int64  `json:"read"`
	Written   int64  `json:"written"`
	Rejected  int64  `json:"rejected"`
	LastRowID string `json:"last_row_id,omitempty"`
}

type Batch struct {
	ID                       string        `json:"id"`
	PreviewID                string        `json:"preview_id"`
	SourceTable              string        `json:"source_table"`
	TargetTable              string        `json:"target_table"`
	InputFingerprint         string        `json:"input_fingerprint"`
	IdempotencyKey           string        `json:"idempotency_key"`
	Status                   BatchStatus   `json:"status"`
	Progress                 BatchProgress `json:"progress"`
	Version                  int64         `json:"version"`
	FailureCode              string        `json:"failure_code,omitempty"`
	InitialTargetRows        int64         `json:"initial_target_rows"`
	FinalTargetRows          int64         `json:"final_target_rows"`
	ReconciliationDifference int64         `json:"reconciliation_difference"`
	RollbackCheckpoint       string        `json:"rollback_checkpoint,omitempty"`
	CreatedBy                string        `json:"created_by"`
	CreatedAt                time.Time     `json:"created_at"`
	UpdatedAt                time.Time     `json:"updated_at"`
}

func NewBatch(id string, preview Preview, sourceTable, targetTable, key, actor string, now time.Time) (Batch, error) {
	if !preview.IsConfirmed() {
		return Batch{}, ErrPreviewRequired
	}
	if sourceTable == targetTable {
		return Batch{}, ErrSourceOverwrite
	}
	if id == "" || key == "" || actor == "" {
		return Batch{}, NewValidationError("batch identity is incomplete")
	}
	return Batch{ID: id, PreviewID: preview.ID, SourceTable: sourceTable, TargetTable: targetTable, InputFingerprint: preview.InputFingerprint, IdempotencyKey: key, Status: BatchPending, Version: 1, CreatedBy: actor, CreatedAt: now, UpdatedAt: now}, nil
}

func (b *Batch) Start(expected int64, now time.Time) error {
	if b.Version != expected {
		return ErrVersionConflict
	}
	if b.Status != BatchPending && b.Status != BatchRecovering {
		return ErrInvalidTransition
	}
	b.Status = BatchRunning
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) RequestCancel(expected int64, now time.Time) error {
	switch b.Status {
	case BatchPending:
		b.Status = BatchCancelling
	case BatchRunning, BatchRecovering:
		b.Status = BatchCancelling
	default:
		return ErrInvalidTransition
	}
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) Fail(expected int64, code string, progress BatchProgress, now time.Time) error {
	if b.Version != expected {
		return ErrVersionConflict
	}
	if b.Status != BatchRunning && b.Status != BatchCancelling {
		return ErrInvalidTransition
	}
	b.Status = BatchFailed
	b.Progress = progress
	b.FailureCode = code
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) Recover(expected int64, fingerprint string, now time.Time) error {
	if b.Version != expected {
		return ErrVersionConflict
	}
	if b.Status != BatchFailed || fingerprint != b.InputFingerprint {
		return ErrInvalidTransition
	}
	b.Status = BatchRecovering
	b.FailureCode = ""
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) Complete(expected, finalTargetRows, difference int64, now time.Time) error {
	if b.Version != expected {
		return ErrVersionConflict
	}
	if b.Status != BatchRunning {
		return ErrInvalidTransition
	}
	if difference != 0 {
		return ErrConflict
	}
	b.Status = BatchCompleted
	b.FinalTargetRows = finalTargetRows
	b.ReconciliationDifference = difference
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) MarkCancelled(expected int64, now time.Time) error {
	b.Status = BatchCancelled
	b.Version++
	b.UpdatedAt = now
	return nil
}

func (b *Batch) MarkRolledBack(expected, finalTargetRows int64, now time.Time) error {
	if b.Version != expected {
		return ErrVersionConflict
	}
	switch b.Status {
	case BatchCompleted, BatchFailed, BatchCancelled:
	default:
		return ErrInvalidTransition
	}
	if finalTargetRows != b.InitialTargetRows {
		return ErrConflict
	}
	b.Status = BatchRolledBack
	b.FinalTargetRows = finalTargetRows
	b.ReconciliationDifference = 0
	b.Version++
	b.UpdatedAt = now
	return nil
}
