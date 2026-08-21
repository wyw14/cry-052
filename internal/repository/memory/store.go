package memory

import (
	"encoding/json"
	"sync"

	"github.com/wyw14/cry052/internal/domain"
)

type Store struct {
	mu          sync.RWMutex
	batchMu     sync.RWMutex
	datasources map[string]domain.DataSource
	tables      map[string]domain.TableSchema
	policies    map[string]domain.PolicyVersion
	previews    map[string]domain.Preview
	batches     map[string]domain.Batch
	idempotency map[string]string
	approvals   map[string]domain.Approval
	audits      []domain.AuditEvent
}

func New() *Store {
	return &Store{
		datasources: map[string]domain.DataSource{},
		tables:      map[string]domain.TableSchema{},
		policies:    map[string]domain.PolicyVersion{},
		previews:    map[string]domain.Preview{},
		batches:     map[string]domain.Batch{},
		idempotency: map[string]string{},
		approvals:   map[string]domain.Approval{},
	}
}

func clone[T any](value T) T {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var copied T
	if err := json.Unmarshal(payload, &copied); err != nil {
		panic(err)
	}
	return copied
}

func filterMatches(filters map[string]string, values map[string]string) bool {
	for key, expected := range filters {
		if values[key] != expected {
			return false
		}
	}
	return true
}
