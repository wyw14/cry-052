package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreateBatch(ctx context.Context, batch domain.Batch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.batches[batch.ID]; exists {
		return domain.ErrConflict
	}
	if _, exists := s.idempotency[batch.IdempotencyKey]; exists {
		return domain.ErrConflict
	}
	s.batches[batch.ID], s.idempotency[batch.IdempotencyKey] = clone(batch), batch.ID
	return nil
}

func (s *Store) GetBatch(ctx context.Context, id string) (domain.Batch, error) {
	if err := ctx.Err(); err != nil {
		return domain.Batch{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.batches[id]
	if !ok {
		return domain.Batch{}, domain.ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) GetBatchByIdempotency(ctx context.Context, key string) (domain.Batch, error) {
	if err := ctx.Err(); err != nil {
		return domain.Batch{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.idempotency[key]
	if !ok {
		return domain.Batch{}, domain.ErrNotFound
	}
	return clone(s.batches[id]), nil
}

func (s *Store) SaveBatch(ctx context.Context, batch domain.Batch, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.batches[batch.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrVersionConflict
	}
	s.batches[batch.ID] = clone(batch)
	return nil
}

func (s *Store) ListBatches(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Batch], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.Batch]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Batch, 0, len(s.batches))
	for _, item := range s.batches {
		if filterMatches(request.Filters, map[string]string{"status": string(item.Status), "created_by": item.CreatedBy}) {
			items = append(items, clone(item))
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
	return domain.Paginate(items, request), nil
}
