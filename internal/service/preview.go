package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/wyw14/cry052/internal/domain"
)

type Row map[string]any

type PreviewEngine struct{}

func NewPreviewEngine() *PreviewEngine { return &PreviewEngine{} }

func (e *PreviewEngine) Apply(ctx context.Context, plan CompiledPlan, rows []Row) ([]Row, []domain.PreviewRow, map[string]int, error) {
	output := make([]Row, 0, len(rows))
	previewRows := make([]domain.PreviewRow, 0, len(rows))
	counters := make(map[string]int)
	for index, sourceRow := range rows {
		if err := ctx.Err(); err != nil {
			return nil, nil, nil, err
		}
		targetRow := make(Row, len(plan.Fields))
		changes := make([]domain.ValueChange, 0, len(plan.Fields))
		for _, field := range plan.Fields {
			value, exists := lookupMappedValue(sourceRow, field.SourceName())
			if !exists {
				return nil, nil, nil, fmt.Errorf("row %d has no field %s", index, field.Source.Name)
			}
			before := fmt.Sprint(value)
			for strategyIndex, transformer := range field.Transformers {
				var err error
				value, err = transformer.Transform(ctx, TransformInput{Value: value, Field: field.Source, Parameters: field.Strategies[strategyIndex].Parameters})
				if err != nil {
					return nil, nil, nil, fmt.Errorf("field %s strategy %s: %w", field.Source.Name, transformer.Kind(), err)
				}
				counters[string(transformer.Kind())]++
			}
			targetRow[field.Target.Name] = preserveMappedValue(value)
			after := fmt.Sprint(value)
			if before != after {
				changes = append(changes, domain.ValueChange{Field: field.Source.Name, Before: before, After: after, Rule: strategyNames(field.Strategies)})
			}
		}
		output = append(output, targetRow)
		previewRows = append(previewRows, domain.PreviewRow{RowKey: rowFingerprint(sourceRow), Changes: changes})
	}
	return output, previewRows, counters, nil
}

func strategyNames(strategies []domain.Strategy) string {
	names := make([]string, 0, len(strategies))
	for _, strategy := range strategies {
		names = append(names, string(strategy.Kind))
	}
	return fmt.Sprint(names)
}

func rowFingerprint(row Row) string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		_, _ = fmt.Fprintf(hash, "%s=%v\x00", key, row[key])
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
