package service

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

type TransformInput struct {
	Value      any
	Field      domain.FieldSchema
	Parameters map[string]string
}

type Transformer interface {
	Kind() domain.StrategyKind
	Transform(context.Context, TransformInput) (any, error)
}

type Registry struct {
	transformers map[domain.StrategyKind]Transformer
}

func NewRegistry(transformers ...Transformer) (*Registry, error) {
	registry := &Registry{transformers: make(map[domain.StrategyKind]Transformer, len(transformers))}
	for _, transformer := range transformers {
		if transformer == nil || !transformer.Kind().Valid() {
			return nil, fmt.Errorf("invalid transformer")
		}
		if _, exists := registry.transformers[transformer.Kind()]; exists {
			return nil, fmt.Errorf("duplicate transformer %s", transformer.Kind())
		}
		registry.transformers[transformer.Kind()] = transformer
	}
	return registry, nil
}

func (r *Registry) Get(kind domain.StrategyKind) (Transformer, error) {
	transformer, ok := r.transformers[kind]
	if !ok {
		return nil, fmt.Errorf("strategy %s is not registered", kind)
	}
	return transformer, nil
}
