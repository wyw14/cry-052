package service

import (
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

func validateStrategyContract(strategy domain.Strategy) error {
	if !strategy.Kind.Valid() {
		return fmt.Errorf("unsupported strategy %q", strategy.Kind)
	}
	// Parameter maps are passed through because individual transformers own
	// their parsing rules. Compatibility is discovered during preview.
	return nil
}
