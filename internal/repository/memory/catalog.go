package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) PutTable(ctx context.Context, table domain.TableSchema) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tables[table.ID]; exists {
		return domain.ErrConflict
	}
	for _, existing := range s.tables {
		if existing.DataSourceID == table.DataSourceID && existing.QualifiedName() == table.QualifiedName() {
			return domain.ErrConflict
		}
	}
	s.tables[table.ID] = clone(table)
	return nil
}

func (s *Store) GetTable(ctx context.Context, id string) (domain.TableSchema, error) {
	if err := ctx.Err(); err != nil {
		return domain.TableSchema{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.tables[id]
	if !ok {
		return domain.TableSchema{}, domain.ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) SaveTable(ctx context.Context, table domain.TableSchema, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.tables[table.ID]
	if !exists {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrVersionConflict
	}
	s.tables[table.ID] = clone(table)
	return nil
}

func (s *Store) ListTables(ctx context.Context, dataSourceID string, request domain.PageRequest) (domain.Page[domain.TableSchema], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.TableSchema]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.TableSchema, 0)
	for _, item := range s.tables {
		if item.DataSourceID == dataSourceID && tableMatches(item, request.Filters) {
			items = append(items, clone(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		switch request.Sort {
		case "version":
			return items[i].Version < items[j].Version
		case "discovered_at":
			return items[i].DiscoveredAt.Before(items[j].DiscoveredAt)
		default:
			return items[i].QualifiedName() < items[j].QualifiedName()
		}
	})
	return domain.Paginate(items, request), nil
}

func tableMatches(table domain.TableSchema, filters map[string]string) bool {
	for key, expected := range filters {
		switch key {
		case "schema":
			if table.Schema != expected {
				return false
			}
		case "name":
			if table.Name != expected {
				return false
			}
		case "sensitivity", "category", "scope":
			matched := false
			for _, field := range table.Fields {
				if key == "sensitivity" && string(field.Sensitivity) == expected {
					matched = true
				}
				if key == "category" && field.Category == expected {
					matched = true
				}
				if key == "scope" {
					for _, scope := range field.Scopes {
						if scope == expected {
							matched = true
						}
					}
				}
			}
			if !matched {
				return false
			}
		}
	}
	return true
}
