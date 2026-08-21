package service

import (
	"context"
	"strings"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestCompilerRejectsIncompatibleStrategyParametersBeforePreview(t *testing.T) {
	registry, err := NewRegistry(MaskTransformer{}, ReplaceTransformer{}, GeneralizeTransformer{}, NewHashTransformer(staticSecretResolver{}), KeepTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	compiler := NewCompiler(registry)
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "public", Name: "people", Version: 1, Fields: []domain.FieldSchema{{Name: "joined_at", DataType: "timestamp", Sensitivity: domain.SensitivityConfidential}}}
	target := domain.TableSchema{ID: "target", DataSourceID: "ds", Schema: "masked", Name: "people", Version: 1, Fields: []domain.FieldSchema{{Name: "joined_month", DataType: "timestamp"}}}
	_, _, err = compiler.Compile(source, target, []domain.FieldMapping{{SourceField: "joined_at", TargetField: "joined_month", Strategies: []domain.Strategy{{Kind: domain.StrategyGeneralize, Parameters: map[string]string{"mode": "number_bucket", "width": "10", "unknown": "value"}}}}})
	if err == nil {
		t.Fatal("incompatible strategy parameters were accepted for preview")
	}
	if !strings.Contains(err.Error(), "strategy contract") {
		t.Fatalf("expected strategy contract error, got %v", err)
	}
}

type staticSecretResolver struct{}

func (staticSecretResolver) ResolveSecret(_ context.Context, _ string) ([]byte, error) {
	return []byte("fixed-secret"), nil
}
