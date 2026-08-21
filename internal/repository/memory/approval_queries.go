package memory

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) ListApprovals(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Approval], error) {
	if err := ctx.Err(); err != nil {
		return domain.Page[domain.Approval]{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Approval, 0, len(s.approvals))
	for _, item := range s.approvals {
		if filterMatches(request.Filters, map[string]string{
			"decision": string(item.Decision), "policy_version_id": item.PolicyVersionID,
			"requested_by": item.RequestedBy, "reviewer": item.Reviewer,
		}) {
			items = append(items, clone(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if request.Sort == "decision" {
			return items[i].Decision < items[j].Decision
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return domain.Paginate(items, request), nil
}
