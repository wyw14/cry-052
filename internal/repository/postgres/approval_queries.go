package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) ListApprovals(ctx context.Context, request domain.PageRequest) (domain.Page[domain.Approval], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM approvals`)
	if err != nil {
		return domain.Page[domain.Approval]{}, err
	}
	defer rows.Close()
	items := make([]domain.Approval, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.Approval]{}, err
		}
		item, err := decode[domain.Approval](payload)
		if err != nil {
			return domain.Page[domain.Approval]{}, err
		}
		if pgFilters(request.Filters, map[string]string{
			"decision": string(item.Decision), "policy_version_id": item.PolicyVersionID,
			"requested_by": item.RequestedBy, "reviewer": item.Reviewer,
		}) {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.Approval]{}, err
	}
	sort.Slice(items, func(i, j int) bool {
		if request.Sort == "decision" {
			return items[i].Decision < items[j].Decision
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return domain.Paginate(items, request), nil
}
