package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreateDataSource(ctx context.Context, item domain.DataSource) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO data_sources(id,name,kind,status,version,created_at,payload) VALUES($1,$2,$3,$4,$5,$6,$7)`, item.ID, item.Name, item.Kind, item.Status, item.Version, item.CreatedAt, payload)
	return translate(err)
}

func (s *Store) GetDataSource(ctx context.Context, id string) (domain.DataSource, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM data_sources WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.DataSource{}, translate(err)
	}
	return decode[domain.DataSource](payload)
}

func (s *Store) SaveDataSource(ctx context.Context, item domain.DataSource, expected int64) error {
	payload, err := encode(item)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE data_sources SET name=$2,kind=$3,status=$4,version=$5,payload=$6 WHERE id=$1 AND version=$7`, item.ID, item.Name, item.Kind, item.Status, item.Version, payload, expected)
	if err != nil {
		return translate(err)
	}
	return changed(tag)
}

func (s *Store) ListDataSources(ctx context.Context, request domain.PageRequest) (domain.Page[domain.DataSource], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM data_sources`)
	if err != nil {
		return domain.Page[domain.DataSource]{}, err
	}
	defer rows.Close()
	items := []domain.DataSource{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.DataSource]{}, err
		}
		item, err := decode[domain.DataSource](payload)
		if err != nil {
			return domain.Page[domain.DataSource]{}, err
		}
		if pgFilters(request.Filters, map[string]string{"kind": string(item.Kind), "status": string(item.Status)}) {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return domain.Page[domain.DataSource]{}, err
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

func pgFilters(filters, values map[string]string) bool {
	for key, value := range filters {
		if values[key] != value {
			return false
		}
	}
	return true
}
