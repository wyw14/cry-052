package service

import (
	"context"

	"github.com/wyw14/cry052/internal/domain"
)

type KeepTransformer struct{}

func (KeepTransformer) Kind() domain.StrategyKind { return domain.StrategyKeep }

func (KeepTransformer) Transform(ctx context.Context, input TransformInput) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return input.Value, nil
}
