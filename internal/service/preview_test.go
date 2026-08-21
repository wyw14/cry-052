package service

import (
	"context"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestPreviewAppliesOrderedMaskAndKeepsSourceRow(t *testing.T) {
	registry, err := NewRegistry(MaskTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	compiler := NewCompiler(registry)
	source := domain.TableSchema{ID: "s", DataSourceID: "d", Schema: "public", Name: "people", Fields: []domain.FieldSchema{{Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted}}, Fingerprint: "fp", Version: 1}
	target := domain.TableSchema{ID: "t", DataSourceID: "d", Schema: "masked", Name: "people", Fields: []domain.FieldSchema{{Name: "mobile", DataType: "text"}}, Fingerprint: "fp2", Version: 1}
	plan, conflicts, err := compiler.Compile(source, target, []domain.FieldMapping{{SourceField: "mobile", TargetField: "mobile", Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "4"}}}}})
	if err != nil || len(conflicts) > 0 {
		t.Fatalf("compile: %v %+v", err, conflicts)
	}
	input := Row{"mobile": "13812345678"}
	output, preview, counters, err := NewPreviewEngine().Apply(context.Background(), plan, []Row{input})
	if err != nil {
		t.Fatal(err)
	}
	if output[0]["mobile"] != "138****5678" {
		t.Fatalf("unexpected masked value %v", output)
	}
	if input["mobile"] != "13812345678" {
		t.Fatal("source row was modified")
	}
	if len(preview[0].Changes) != 1 || counters["mask"] != 1 {
		t.Fatalf("missing evidence: %+v %+v", preview, counters)
	}
}
