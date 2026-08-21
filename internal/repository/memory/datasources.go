package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreateDataSource(ctx context.Context, item domain.DataSource) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.datasources {
		if existing.Name == item.Name {
			return domain.ErrConflict
		}
	}
	if _, exists := s.datasources[item.ID]; exists {
		return domain.ErrConflict
	}
	s.datasources[item.ID] = clone(item)
	return nil
}

func (s *Store) GetDataSource(ctx context.Context, id string) (domain.DataSource, error) {
	if err := ctx.Err(); err != nil {
		return domain.DataSource{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.datasources[id]
	if !ok {
		return domain.DataSource{}, domain.ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) SaveDataSource(ctx context.Context, item domain.DataSource, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.datasources[item.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrVersionConflict
	}
	s.datasources[item.ID] = clone(item)
	return nil
}

func (s *Store) ListDataSources(ctx context.Context, request domain.PageRequest) (domain.Page[domain.DataSource], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.DataSource]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.DataSource, 0, len(s.datasources))
	for _, item := range s.datasources {
		if filterMatches(request.Filters, map[string]string{"kind": string(item.Kind), "status": string(item.Status)}) {
			items = append(items, clone(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		switch request.Sort {
		case "name":
			return items[i].Name < items[j].Name
		case "status":
			return items[i].Status < items[j].Status
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	return domain.Paginate(items, request), nil
}
