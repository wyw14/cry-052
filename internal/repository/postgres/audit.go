package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) AppendAudit(ctx context.Context, item domain.AuditEvent) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO audit_events(id,request_id,actor,action,resource,outcome,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, item.ID, item.RequestID, item.Actor, item.Action, item.Resource, item.Outcome, item.CreatedAt, payload)
	return translate(err)
}
func (s *Store) ListAudit(ctx context.Context, request domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM audit_events`)
	if err != nil {
		return domain.Page[domain.AuditEvent]{}, err
	}
	defer rows.Close()
	items := []domain.AuditEvent{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.AuditEvent]{}, err
		}
		item, err := decode[domain.AuditEvent](payload)
		if err != nil {
			return domain.Page[domain.AuditEvent]{}, err
		}
		if pgFilters(request.Filters, map[string]string{"actor": item.Actor, "action": item.Action, "resource": item.Resource, "outcome": item.Outcome}) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if request.Sort == "action" {
			return items[i].Action < items[j].Action
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return domain.Paginate(items, request), rows.Err()
}
