package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreatePolicy(ctx context.Context, policy domain.PolicyVersion) error {
	payload, err := encode(policy)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO policy_versions(id,group_id,version,status,revision,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, policy.ID, policy.GroupID, policy.Version, policy.Status, policy.Revision, policy.CreatedAt, payload)
	return translate(err)
}

func (s *Store) GetPolicy(ctx context.Context, id string) (domain.PolicyVersion, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM policy_versions WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.PolicyVersion{}, translate(err)
	}
	return decode[domain.PolicyVersion](payload)
}

func (s *Store) SavePolicy(ctx context.Context, policy domain.PolicyVersion, expected int64) error {
	payload, err := encode(policy)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE policy_versions SET status=$2,revision=$3,payload=$4 WHERE id=$1 AND revision=$5`, policy.ID, policy.Status, policy.Revision, payload, expected)
	if err != nil {
		return translate(err)
	}
	return changed(tag)
}

func (s *Store) ListPolicies(ctx context.Context, request domain.PageRequest) (domain.Page[domain.PolicyVersion], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM policy_versions`)
	if err != nil {
		return domain.Page[domain.PolicyVersion]{}, err
	}
	defer rows.Close()
	items := []domain.PolicyVersion{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.PolicyVersion]{}, err
		}
		item, err := decode[domain.PolicyVersion](payload)
		if err != nil {
			return domain.Page[domain.PolicyVersion]{}, err
		}
		scope := ""
		if expected, ok := request.Filters["scope"]; ok {
			for _, candidate := range item.Scopes {
				if candidate == expected {
					scope = expected
					break
				}
			}
		}
		if pgFilters(request.Filters, map[string]string{"status": string(item.Status), "group_id": item.GroupID, "scope": scope}) {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.PolicyVersion]{}, err
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
