package memory

import (
	"context"

	"github.com/wyw14/cry052/internal/persistence"
)

func (s *Store) ApplyMutation(ctx context.Context, mutation persistence.Mutation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx := &Store{
		datasources: clone(s.datasources), tables: clone(s.tables), policies: clone(s.policies), previews: clone(s.previews),
		batches: clone(s.batches), idempotency: clone(s.idempotency), approvals: clone(s.approvals), audits: clone(s.audits),
	}
	if mutation.CreateDataSource != nil {
		if err := tx.CreateDataSource(ctx, *mutation.CreateDataSource); err != nil {
			return err
		}
	}
	if mutation.UpdateDataSource != nil {
		if err := tx.SaveDataSource(ctx, mutation.UpdateDataSource.Value, mutation.UpdateDataSource.Expected); err != nil {
			return err
		}
	}
	if mutation.CreateTable != nil {
		if err := tx.PutTable(ctx, *mutation.CreateTable); err != nil {
			return err
		}
	}
	if mutation.UpdateTable != nil {
		if err := tx.SaveTable(ctx, mutation.UpdateTable.Value, mutation.UpdateTable.Expected); err != nil {
			return err
		}
	}
	if mutation.CreatePolicy != nil {
		if err := tx.CreatePolicy(ctx, *mutation.CreatePolicy); err != nil {
			return err
		}
	}
	if mutation.UpdatePolicy != nil {
		if err := tx.SavePolicy(ctx, mutation.UpdatePolicy.Value, mutation.UpdatePolicy.Expected); err != nil {
			return err
		}
	}
	if mutation.CreateApproval != nil {
		if err := tx.CreateApproval(ctx, *mutation.CreateApproval); err != nil {
			return err
		}
	}
	if mutation.UpdateApproval != nil {
		if err := tx.SaveApproval(ctx, mutation.UpdateApproval.Value, mutation.UpdateApproval.Expected); err != nil {
			return err
		}
	}
	if mutation.CreatePreview != nil {
		if err := tx.CreatePreview(ctx, *mutation.CreatePreview); err != nil {
			return err
		}
	}
	if mutation.UpdatePreview != nil {
		if err := tx.SavePreview(ctx, *mutation.UpdatePreview); err != nil {
			return err
		}
	}
	if mutation.CreateBatch != nil {
		if err := tx.CreateBatch(ctx, *mutation.CreateBatch); err != nil {
			return err
		}
	}
	if mutation.UpdateBatch != nil {
		if err := tx.SaveBatch(ctx, mutation.UpdateBatch.Value, mutation.UpdateBatch.Expected); err != nil {
			return err
		}
	}
	if err := tx.AppendAudit(ctx, mutation.Audit); err != nil {
		return err
	}
	s.datasources, s.tables, s.policies, s.previews = tx.datasources, tx.tables, tx.policies, tx.previews
	s.batches, s.idempotency, s.approvals, s.audits = tx.batches, tx.idempotency, tx.approvals, tx.audits
	return nil
}
