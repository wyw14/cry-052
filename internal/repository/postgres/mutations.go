package postgres

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

func (s *Store) ApplyMutation(ctx context.Context, mutation persistence.Mutation) error {
	if err := validateMutationShape(mutation); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if item := mutation.CreateDataSource; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO data_sources(id,name,kind,status,version,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, item.ID, item.Name, item.Kind, item.Status, item.Version, item.CreatedAt, payload); err != nil {
			return translate(err)
		}
	}
	if update := mutation.UpdateDataSource; update != nil {
		item := update.Value
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE data_sources SET name=$2,kind=$3,status=$4,version=$5,payload=$6 WHERE id=$1 AND version=$7`, item.ID, item.Name, item.Kind, item.Status, item.Version, payload, update.Expected)
		if err != nil {
			return translate(err)
		}
		if err := changed(tag); err != nil {
			return err
		}
	}
	if item := mutation.CreateTable; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO table_schemas(id,datasource_id,qualified_name,fingerprint,payload) VALUES($1,$2,$3,$4,$5)`, item.ID, item.DataSourceID, item.QualifiedName(), item.Fingerprint, payload); err != nil {
			return translate(err)
		}
	}
	if update := mutation.UpdateTable; update != nil {
		item := update.Value
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE table_schemas SET fingerprint=$2,payload=$3 WHERE id=$1 AND (payload->>'version')::bigint=$4`, item.ID, item.Fingerprint, payload, update.Expected)
		if err != nil {
			return translate(err)
		}
		if err := changed(tag); err != nil {
			return err
		}
	}
	if item := mutation.CreatePolicy; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO policy_versions(id,group_id,version,status,revision,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, item.ID, item.GroupID, item.Version, item.Status, item.Revision, item.CreatedAt, payload); err != nil {
			return translate(err)
		}
	}
	if update := mutation.UpdatePolicy; update != nil {
		item := update.Value
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE policy_versions SET status=$2,revision=$3,payload=$4 WHERE id=$1 AND revision=$5`, item.ID, item.Status, item.Revision, payload, update.Expected)
		if err != nil {
			return translate(err)
		}
		if err := changed(tag); err != nil {
			return err
		}
	}
	if item := mutation.CreateApproval; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO approvals(id,policy_version_id,decision,revision,payload) VALUES($1,$2,$3,$4,$5)`, item.ID, item.PolicyVersionID, item.Decision, item.Revision, payload); err != nil {
			return translate(err)
		}
	}
	if update := mutation.UpdateApproval; update != nil {
		item := update.Value
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE approvals SET decision=$2,revision=$3,payload=$4 WHERE id=$1 AND revision=$5`, item.ID, item.Decision, item.Revision, payload, update.Expected)
		if err != nil {
			return translate(err)
		}
		if err := changed(tag); err != nil {
			return err
		}
	}
	if item := mutation.CreatePreview; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO previews(id,source_table_id,target_table_id,policy_version_id,confirmed_at,payload) VALUES($1,$2,$3,$4,$5,$6)`, item.ID, item.SourceTableID, item.TargetTableID, item.PolicyVersionID, item.ConfirmedAt, payload); err != nil {
			return translate(err)
		}
	}
	if item := mutation.UpdatePreview; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE previews SET confirmed_at=$2,payload=$3 WHERE id=$1`, item.ID, item.ConfirmedAt, payload)
		if err != nil {
			return translate(err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
	}
	if item := mutation.CreateBatch; item != nil {
		payload, err := encode(item)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO batches(id,preview_id,idempotency_key,status,version,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, item.ID, item.PreviewID, item.IdempotencyKey, item.Status, item.Version, item.CreatedAt, payload); err != nil {
			return translate(err)
		}
	}
	if update := mutation.UpdateBatch; update != nil {
		item := update.Value
		payload, err := encode(item)
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE batches SET status=$2,version=$3,payload=$4 WHERE id=$1 AND version=$5`, item.ID, item.Status, item.Version, payload, update.Expected)
		if err != nil {
			return translate(err)
		}
		if err := changed(tag); err != nil {
			return err
		}
	}
	audit := mutation.Audit
	payload, err := encode(audit)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(id,request_id,actor,action,resource,outcome,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, audit.ID, audit.RequestID, audit.Actor, audit.Action, audit.Resource, audit.Outcome, audit.CreatedAt, payload); err != nil {
		return translate(err)
	}
	return tx.Commit(ctx)
}

func validateMutationShape(mutation persistence.Mutation) error {
	if mutation.UpdatePolicy != nil && mutation.CreateApproval != nil &&
		mutation.UpdatePolicy.Value.ID != mutation.CreateApproval.PolicyVersionID {
		return fmt.Errorf("policy submission mutation links different policy records: %w", domain.ErrConflict)
	}
	return nil
}
