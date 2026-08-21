package memory

import (
	"context"

	"github.com/wyw14/cry052/internal/persistence"
)

func (s *Store) ApplyMutation(ctx context.Context, mutation persistence.Mutation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if mutation.CreateDataSource != nil {
		if err := s.CreateDataSource(ctx, *mutation.CreateDataSource); err != nil {
			return err
		}
	}
	if mutation.UpdateDataSource != nil {
		if err := s.SaveDataSource(ctx, mutation.UpdateDataSource.Value, mutation.UpdateDataSource.Expected); err != nil {
			return err
		}
	}
	if mutation.CreateTable != nil {
		if err := s.PutTable(ctx, *mutation.CreateTable); err != nil {
			return err
		}
	}
	if mutation.UpdateTable != nil {
		if err := s.SaveTable(ctx, mutation.UpdateTable.Value, mutation.UpdateTable.Expected); err != nil {
			return err
		}
	}
	if mutation.CreatePolicy != nil {
		if err := s.CreatePolicy(ctx, *mutation.CreatePolicy); err != nil {
			return err
		}
	}
	if mutation.UpdatePolicy != nil {
		if err := s.SavePolicy(ctx, mutation.UpdatePolicy.Value, mutation.UpdatePolicy.Expected); err != nil {
			return err
		}
	}
	if mutation.CreateApproval != nil {
		if err := s.CreateApproval(ctx, *mutation.CreateApproval); err != nil {
			return err
		}
	}
	if mutation.UpdateApproval != nil {
		if err := s.SaveApproval(ctx, mutation.UpdateApproval.Value, mutation.UpdateApproval.Expected); err != nil {
			return err
		}
	}
	if mutation.CreatePreview != nil {
		if err := s.CreatePreview(ctx, *mutation.CreatePreview); err != nil {
			return err
		}
	}
	if mutation.UpdatePreview != nil {
		if err := s.SavePreview(ctx, *mutation.UpdatePreview); err != nil {
			return err
		}
	}
	if mutation.CreateBatch != nil {
		if err := s.CreateBatch(ctx, *mutation.CreateBatch); err != nil {
			return err
		}
	}
	if mutation.UpdateBatch != nil {
		if err := s.SaveBatch(ctx, mutation.UpdateBatch.Value, mutation.UpdateBatch.Expected); err != nil {
			return err
		}
	}
	if err := s.AppendAudit(ctx, mutation.Audit); err != nil {
		return err
	}
	return nil
}
