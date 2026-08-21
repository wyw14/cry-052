package platform

import (
	"context"
	"fmt"
	"sync"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/service"
)

type SampleDatabase struct {
	mu         sync.RWMutex
	tables     map[string][]service.Row
	operations map[string]int64
}

func NewSampleDatabase() *SampleDatabase {
	return &SampleDatabase{tables: make(map[string][]service.Row), operations: make(map[string]int64)}
}

func (d *SampleDatabase) Seed(table string, rows []service.Row) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tables[table] = cloneRows(rows)
}

func (d *SampleDatabase) Rows(ctx context.Context, table domain.TableSchema, limit int) ([]service.Row, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, ok := d.tables[table.QualifiedName()]
	if !ok {
		return nil, fmt.Errorf("sample table %s is not registered", table.QualifiedName())
	}
	if limit > len(rows) {
		limit = len(rows)
	}
	return cloneRows(rows[:limit]), nil
}

func (d *SampleDatabase) ReadChunk(ctx context.Context, table, cursor string, size int) ([]service.Row, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, ok := d.tables[table]
	if !ok {
		return nil, "", fmt.Errorf("table %s is not registered", table)
	}
	start := 0
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "%d", &start); err != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
	}
	if start >= len(rows) {
		return nil, cursor, nil
	}
	end := start + size
	if end > len(rows) {
		end = len(rows)
	}
	return cloneRows(rows[start:end]), fmt.Sprint(end), nil
}

func (d *SampleDatabase) WriteChunk(ctx context.Context, table, operationID string, rows []service.Row) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if operationID == "" {
		return 0, fmt.Errorf("operation id is required")
	}
	operationKey := table + "\x00" + operationID
	if written, exists := d.operations[operationKey]; exists {
		return written, nil
	}
	d.tables[table] = append(d.tables[table], cloneRows(rows)...)
	written := int64(len(rows))
	d.operations[operationKey] = written
	return written, nil
}

func (d *SampleDatabase) Count(ctx context.Context, table string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, ok := d.tables[table]
	if !ok {
		return 0, fmt.Errorf("table %s is not registered", table)
	}
	return int64(len(rows)), nil
}

func (d *SampleDatabase) Rollback(ctx context.Context, table, checkpoint string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var keep int
	if _, err := fmt.Sscanf(checkpoint, "%d", &keep); err != nil || keep < 0 || keep > len(d.tables[table]) {
		return fmt.Errorf("invalid rollback checkpoint")
	}
	d.tables[table] = append([]service.Row(nil), d.tables[table][:keep]...)
	return nil
}

func cloneRows(rows []service.Row) []service.Row {
	copied := make([]service.Row, len(rows))
	for i, row := range rows {
		copied[i] = make(service.Row, len(row))
		for key, value := range row {
			copied[i][key] = value
		}
	}
	return copied
}
