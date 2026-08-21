package postgres

import (
	"context"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) PutTable(ctx context.Context, table domain.TableSchema) error {
	payload, err := encode(table)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO table_schemas(id,datasource_id,qualified_name,fingerprint,payload) VALUES($1,$2,$3,$4,$5)`, table.ID, table.DataSourceID, table.QualifiedName(), table.Fingerprint, payload)
	return translate(err)
}

func (s *Store) GetTable(ctx context.Context, id string) (domain.TableSchema, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM table_schemas WHERE id=$1`, id).Scan(&payload); err != nil {
		return domain.TableSchema{}, translate(err)
	}
	return decode[domain.TableSchema](payload)
}

func (s *Store) SaveTable(ctx context.Context, table domain.TableSchema, expected int64) error {
	payload, err := encode(table)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE table_schemas SET fingerprint=$2,payload=$3 WHERE id=$1 AND (payload->>'version')::bigint=$4`, table.ID, table.Fingerprint, payload, expected)
	if err != nil {
		return translate(err)
	}
	return changed(tag)
}

func (s *Store) ListTables(ctx context.Context, dataSourceID string, request domain.PageRequest) (domain.Page[domain.TableSchema], error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM table_schemas WHERE datasource_id=$1`, dataSourceID)
	if err != nil {
		return domain.Page[domain.TableSchema]{}, err
	}
	defer rows.Close()
	items := []domain.TableSchema{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.Page[domain.TableSchema]{}, err
		}
		item, err := decode[domain.TableSchema](payload)
		if err != nil {
			return domain.Page[domain.TableSchema]{}, err
		}
		if tableMatches(item, request.Filters) {
			items = append(items, item)
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
	return domain.Paginate(items, request), rows.Err()
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
