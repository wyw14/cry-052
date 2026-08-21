package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/wyw14/cry052/internal/domain"
)

type ReplaceTransformer struct{}

func (ReplaceTransformer) Kind() domain.StrategyKind { return domain.StrategyReplace }

func (ReplaceTransformer) Transform(ctx context.Context, input TransformInput) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value := fmt.Sprint(input.Value)
	mode := input.Parameters["mode"]
	switch mode {
	case "constant":
		if _, ok := input.Parameters["value"]; !ok {
			return nil, fmt.Errorf("constant replacement requires value")
		}
		return input.Parameters["value"], nil
	case "dictionary":
		for _, pair := range strings.Split(input.Parameters["entries"], ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == value {
				return strings.TrimSpace(parts[1]), nil
			}
		}
		if fallback, ok := input.Parameters["fallback"]; ok {
			return fallback, nil
		}
		return nil, fmt.Errorf("dictionary has no replacement for value")
	default:
		return nil, fmt.Errorf("unsupported replacement mode %q", mode)
	}
}
