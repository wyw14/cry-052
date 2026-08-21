package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, clone(event))
	return nil
}

func (s *Store) ListAudit(ctx context.Context, request domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.AuditEvent]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.AuditEvent, 0, len(s.audits))
	for _, item := range s.audits {
		if filterMatches(request.Filters, map[string]string{"actor": item.Actor, "action": item.Action, "resource": item.Resource, "outcome": item.Outcome}) {
			items = append(items, clone(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if request.Sort == "action" {
			return items[i].Action < items[j].Action
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return domain.Paginate(items, request), nil
}
