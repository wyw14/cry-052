package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wyw14/cry052/internal/domain"
)

type CompiledField struct {
	Source       domain.FieldSchema
	Target       domain.FieldSchema
	Transformers []Transformer
	Strategies   []domain.Strategy
}

type CompiledPlan struct {
	SourceTable domain.TableSchema
	TargetTable domain.TableSchema
	Fields      []CompiledField
}

type Compiler struct {
	registry *Registry
}

func NewCompiler(registry *Registry) *Compiler { return &Compiler{registry: registry} }

func (c *Compiler) Compile(source, target domain.TableSchema, mappings []domain.FieldMapping) (CompiledPlan, []domain.MappingConflict, error) {
	if err := source.Validate(); err != nil {
		return CompiledPlan{}, nil, fmt.Errorf("source schema: %w", err)
	}
	if err := target.Validate(); err != nil {
		return CompiledPlan{}, nil, fmt.Errorf("target schema: %w", err)
	}
	conflicts := domain.DetectMappingConflicts(source, target, mappings)
	if len(conflicts) > 0 {
		return CompiledPlan{}, conflicts, nil
	}
	fields := make([]CompiledField, 0, len(mappings))
	for _, mapping := range mappings {
		sourceField, _ := source.Field(mapping.SourceField)
		targetField, _ := target.Field(mapping.TargetField)
		compiled := CompiledField{Source: sourceField, Target: targetField, Strategies: append([]domain.Strategy(nil), mapping.Strategies...)}
		seen := make(map[domain.StrategyKind]struct{}, len(mapping.Strategies))
		for _, strategy := range mapping.Strategies {
			if _, exists := seen[strategy.Kind]; exists {
				return CompiledPlan{}, nil, fmt.Errorf("field %s repeats strategy %s", mapping.SourceField, strategy.Kind)
			}
			seen[strategy.Kind] = struct{}{}
			transformer, err := c.registry.Get(strategy.Kind)
			if err != nil {
				return CompiledPlan{}, nil, err
			}
			compiled.Transformers = append(compiled.Transformers, transformer)
		}
		if _, keep := seen[domain.StrategyKeep]; keep && len(seen) > 1 {
			return CompiledPlan{}, nil, fmt.Errorf("keep cannot be combined on field %s", mapping.SourceField)
		}
		fields = append(fields, compiled)
	}
	sort.Slice(fields, func(i, j int) bool {
		return strings.ToLower(fields[i].Target.Name) < strings.ToLower(fields[j].Target.Name)
	})
	return CompiledPlan{SourceTable: source, TargetTable: target, Fields: fields}, nil, nil
}
