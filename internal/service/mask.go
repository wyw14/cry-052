package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wyw14/cry052/internal/domain"
)

type MaskTransformer struct{}

func (MaskTransformer) Kind() domain.StrategyKind { return domain.StrategyMask }

func (MaskTransformer) Transform(ctx context.Context, input TransformInput) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, ok := input.Value.(string)
	if !ok {
		return nil, fmt.Errorf("mask requires string input")
	}
	prefix, err := positiveInt(input.Parameters, "prefix", 0)
	if err != nil {
		return nil, err
	}
	suffix, err := positiveInt(input.Parameters, "suffix", 0)
	if err != nil {
		return nil, err
	}
	runes := []rune(value)
	if prefix+suffix >= len(runes) {
		return strings.Repeat("*", utf8.RuneCountInString(value)), nil
	}
	masked := append([]rune(nil), runes[:prefix]...)
	masked = append(masked, []rune(strings.Repeat("*", len(runes)-prefix-suffix))...)
	masked = append(masked, runes[len(runes)-suffix:]...)
	return string(masked), nil
}

func positiveInt(parameters map[string]string, key string, fallback int) (int, error) {
	raw, ok := parameters[key]
	if !ok || raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}
