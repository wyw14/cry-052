package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

type CreateBatch struct{ PreviewID, IdempotencyKey string }

func (a *App) CreateBatch(ctx context.Context, actor Actor, command CreateBatch) (domain.Batch, error) {
	if err := actor.Require("data_admin"); err != nil {
		return domain.Batch{}, err
	}
	preview, err := a.store.GetPreview(ctx, command.PreviewID)
	if err != nil {
		return domain.Batch{}, err
	}
	source, err := a.store.GetTable(ctx, preview.SourceTableID)
	if err != nil {
		return domain.Batch{}, err
	}
	target, err := a.store.GetTable(ctx, preview.TargetTableID)
	if err != nil {
		return domain.Batch{}, err
	}
	if existing, err := a.store.GetBatchByIdempotency(ctx, command.IdempotencyKey); err == nil {
		if existing.PreviewID == preview.ID && existing.InputFingerprint == preview.InputFingerprint {
			return existing, nil
		}
		return domain.Batch{}, domain.ErrIdempotencyReuse
	}
	batch, err := domain.NewBatch(a.newID(), preview, source.QualifiedName(), target.QualifiedName(), command.IdempotencyKey, actor.ID, a.now())
	if err != nil {
		return domain.Batch{}, err
	}
	audit := a.newAuditEvent(actor, "batch.create", "batch/"+batch.ID, "success", map[string]string{"preview_id": preview.ID})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{CreateBatch: &batch, Audit: audit}); err != nil {
		return domain.Batch{}, fmt.Errorf("create batch: %w", err)
	}
	return batch, nil
}

func (a *App) RunBatch(ctx context.Context, actor Actor, batchID string, mappings []domain.FieldMapping) error {
	batch, err := a.store.GetBatch(ctx, batchID)
	if err != nil {
		return err
	}
	preview, err := a.store.GetPreview(ctx, batch.PreviewID)
	if err != nil {
		return err
	}
	source, err := a.store.GetTable(ctx, preview.SourceTableID)
	if err != nil {
		return err
	}
	target, err := a.store.GetTable(ctx, preview.TargetTableID)
	if err != nil {
		return err
	}
	plan, _, err := a.compiler.Compile(source, target, preview.Mappings)
	if err != nil {
		return err
	}
	if err := a.processor.Run(ctx, batch.ID, plan); err != nil {
		return err
	}
	return nil
}

func (a *App) RollbackBatch(ctx context.Context, actor Actor, batchID string) (domain.Batch, error) {
	batch, err := a.processor.Rollback(ctx, batchID)
	if err != nil {
		return domain.Batch{}, err
	}
	return batch, nil
}

func (a *App) CancelBatch(ctx context.Context, actor Actor, batchID string, expected int64) error {
	if err := actor.Require("data_admin"); err != nil {
		return err
	}
	batch, err := a.store.GetBatch(ctx, batchID)
	if err != nil {
		return err
	}
	previous := batch.Version
	if err := batch.RequestCancel(expected, a.now()); err != nil {
		return err
	}
	audit := a.newAuditEvent(actor, "batch.cancel", "batch/"+batch.ID, "success", nil)
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{UpdateBatch: &persistence.Versioned[domain.Batch]{Value: batch, Expected: previous}, Audit: audit}); err != nil {
		return err
	}
	if batch.Status == domain.BatchCancelling {
		if err := a.processor.Cancel(ctx, batch.ID); err != nil && err != domain.ErrNotFound {
			return err
		}
	}
	return nil
}

func (a *App) RecoverBatch(ctx context.Context, actor Actor, batchID, fingerprint string, expected int64) (domain.Batch, error) {
	if err := actor.Require("data_admin"); err != nil {
		return domain.Batch{}, err
	}
	batch, err := a.store.GetBatch(ctx, batchID)
	if err != nil {
		return domain.Batch{}, err
	}
	previous := batch.Version
	if err := batch.Recover(expected, fingerprint, a.now()); err != nil {
		return domain.Batch{}, err
	}
	audit := a.newAuditEvent(actor, "batch.recover", "batch/"+batch.ID, "success", map[string]string{"checkpoint": batch.Progress.LastRowID})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{UpdateBatch: &persistence.Versioned[domain.Batch]{Value: batch, Expected: previous}, Audit: audit}); err != nil {
		return domain.Batch{}, err
	}
	return batch, nil
}

func (a *App) ListBatches(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.Batch], error) {
	if err := actor.Require("data_admin", "auditor", "masking_executor"); err != nil {
		return domain.Page[domain.Batch]{}, err
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "status": {}, "progress": {}}, map[string]struct{}{"status": {}, "created_by": {}})
	if err != nil {
		return domain.Page[domain.Batch]{}, err
	}
	return a.store.ListBatches(ctx, normalized)
}
