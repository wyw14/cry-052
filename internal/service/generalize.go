package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

type GeneralizeTransformer struct{}

func (GeneralizeTransformer) Kind() domain.StrategyKind { return domain.StrategyGeneralize }

func (GeneralizeTransformer) Transform(ctx context.Context, input TransformInput) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch input.Parameters["mode"] {
	case "number_bucket":
		width, err := strconv.ParseFloat(input.Parameters["width"], 64)
		if err != nil || width <= 0 {
			return nil, fmt.Errorf("number bucket width must be positive")
		}
		value, err := numeric(input.Value)
		if err != nil {
			return nil, err
		}
		lower := math.Floor(value/width) * width
		return fmt.Sprintf("%g-%g", lower, lower+width), nil
	case "date_month":
		value, err := parseTime(input.Value)
		if err != nil {
			return nil, err
		}
		return value.Format("2006-01"), nil
	case "date_year":
		value, err := parseTime(input.Value)
		if err != nil {
			return nil, err
		}
		return value.Format("2006"), nil
	default:
		return nil, fmt.Errorf("unsupported generalization mode")
	}
}

func numeric(value any) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		return strconv.ParseFloat(typed, 64)
	default:
		return 0, fmt.Errorf("value is not numeric")
	}
}

func parseTime(value any) (time.Time, error) {
	if typed, ok := value.(time.Time); ok {
		return typed, nil
	}
	if typed, ok := value.(string); ok {
		return time.Parse(time.RFC3339, typed)
	}
	return time.Time{}, fmt.Errorf("value is not a timestamp")
}
