package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreatePolicy(ctx context.Context, policy domain.PolicyVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.policies[policy.ID]; exists {
		return domain.ErrConflict
	}
	for _, existing := range s.policies {
		if existing.GroupID == policy.GroupID && existing.Version == policy.Version {
			return domain.ErrConflict
		}
	}
	s.policies[policy.ID] = clone(policy)
	return nil
}

func (s *Store) GetPolicy(ctx context.Context, id string) (domain.PolicyVersion, error) {
	if err := ctx.Err(); err != nil {
		return domain.PolicyVersion{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.policies[id]
	if !ok {
		return domain.PolicyVersion{}, domain.ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) SavePolicy(ctx context.Context, policy domain.PolicyVersion, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.policies[policy.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Revision != expected {
		return domain.ErrVersionConflict
	}
	s.policies[policy.ID] = clone(policy)
	return nil
}

func (s *Store) ListPolicies(ctx context.Context, request domain.PageRequest) (domain.Page[domain.PolicyVersion], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.PolicyVersion]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.PolicyVersion, 0, len(s.policies))
	for _, item := range s.policies {
		scope := ""
		if expected, ok := request.Filters["scope"]; ok {
			for _, candidate := range item.Scopes {
				if candidate == expected {
					scope = expected
					break
				}
			}
		}
		values := map[string]string{"status": string(item.Status), "group_id": item.GroupID, "scope": scope}
		if filterMatches(request.Filters, values) {
			items = append(items, clone(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		switch request.Sort {
		case "name":
			return items[i].Name < items[j].Name
		case "version":
			return items[i].Version < items[j].Version
		default:
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
	})
	return domain.Paginate(items, request), nil
}
