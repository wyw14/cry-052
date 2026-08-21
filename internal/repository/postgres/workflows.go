package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreatePreview(ctx context.Context, item domain.Preview) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO previews(id,source_table_id,target_table_id,policy_version_id,confirmed_at,payload) VALUES($1,$2,$3,$4,$5,$6)`, item.ID, item.SourceTableID, item.TargetTableID, item.PolicyVersionID, item.ConfirmedAt, payload)
	return translate(err)
}
func (s *Store) GetPreview(ctx context.Context, id string) (domain.Preview, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM previews WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.Preview{}, translate(err)
	}
	return decode[domain.Preview](payload)
}
func (s *Store) SavePreview(ctx context.Context, item domain.Preview) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE previews SET confirmed_at=$2,payload=$3 WHERE id=$1`, item.ID, item.ConfirmedAt, payload)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) CreateBatch(ctx context.Context, item domain.Batch) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO batches(id,preview_id,idempotency_key,status,version,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, item.ID, item.PreviewID, item.IdempotencyKey, item.Status, item.Version, item.CreatedAt, payload)
	return translate(err)
}
func (s *Store) GetBatch(ctx context.Context, id string) (domain.Batch, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM batches WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.Batch{}, translate(err)
	}
	return decode[domain.Batch](payload)
}
func (s *Store) GetBatchByIdempotency(ctx context.Context, key string) (domain.Batch, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM batches WHERE idempotency_key=$1`, key).Scan(&payload); err != nil {
		return domain.Batch{}, translate(err)
	}
	return decode[domain.Batch](payload)
}
func (s *Store) SaveBatch(ctx context.Context, item domain.Batch, expected int64) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE batches SET status=$2,version=$3,payload=$4 WHERE id=$1 AND version=$5`, item.ID, item.Status, item.Version, payload, expected)
	if err != nil {
		return translate(err)
	}
	return changed(tag)
}
func (s *Store) ListBatches(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Batch], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM batches`)
	if err != nil {
		return domain.Page[domain.Batch]{}, err
	}
	defer rows.Close()
	items := []domain.Batch{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.Batch]{}, err
		}
		item, err := decode[domain.Batch](payload)
		if err != nil {
			return domain.Page[domain.Batch]{}, err
		}
		if pgFilters(request.Filters, map[string]string{"status": string(item.Status), "created_by": item.CreatedBy}) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		switch request.Sort {
		case "status":
			return items[i].Status < items[j].Status
		case "progress":
			return items[i].Progress.Read < items[j].Progress.Read
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	return domain.Paginate(items, request), rows.Err()
}

func (s *Store) CreateApproval(ctx context.Context, item domain.Approval) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO approvals(id,policy_version_id,decision,revision,payload) VALUES($1,$2,$3,$4,$5)`, item.ID, item.PolicyVersionID, item.Decision, item.Revision, payload)
	return translate(err)
}
func (s *Store) GetApproval(ctx context.Context, id string) (domain.Approval, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM approvals WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.Approval{}, translate(err)
	}
	return decode[domain.Approval](payload)
}
func (s *Store) SaveApproval(ctx context.Context, item domain.Approval, expected int64) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE approvals SET decision=$2,revision=$3,payload=$4 WHERE id=$1 AND revision=$5`, item.ID, item.Decision, item.Revision, payload, expected)
	if err != nil {
		return translate(err)
	}
	return changed(tag)
}
