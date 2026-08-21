package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestPreviewResolvesCatalogFieldNamesWithoutMutatingSourceRows(t *testing.T) {
	registry, err := NewRegistry(KeepTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	compiler := NewCompiler(registry)
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "public", Name: "profiles", Version: 1, Fields: []domain.FieldSchema{{Name: "Tags", DataType: "text", Sensitivity: domain.SensitivityInternal}}}
	target := domain.TableSchema{ID: "target", DataSourceID: "ds", Schema: "masked", Name: "profiles", Version: 1, Fields: []domain.FieldSchema{{Name: "labels", DataType: "text"}}}
	plan, conflicts, err := compiler.Compile(source, target, []domain.FieldMapping{{SourceField: "tags", TargetField: "labels", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}})
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("compile mapping: conflicts=%v err=%v", conflicts, err)
	}
	originalTags := []string{"restricted", "analytics"}
	rows := []Row{{"tags": originalTags}}
	output, _, _, err := NewPreviewEngine().Apply(context.Background(), plan, rows)
	if err != nil {
		t.Fatalf("apply preview: %v", err)
	}
	labels, ok := output[0]["labels"].([]string)
	if !ok {
		t.Fatalf("expected slice value, got %#v", output[0]["labels"])
	}
	labels[0] = "changed"
	if !reflect.DeepEqual(rows[0]["tags"], []string{"restricted", "analytics"}) {
		t.Fatalf("preview output mutated source row: %#v", rows[0]["tags"])
	}
}
