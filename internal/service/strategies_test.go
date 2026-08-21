package service

import (
	"context"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestStrategyMatrixAndConflictDetection(t *testing.T) {
	vault := staticSecret{value: []byte("0123456789abcdef")}
	registry, err := NewRegistry(MaskTransformer{}, ReplaceTransformer{}, GeneralizeTransformer{}, NewHashTransformer(vault), KeepTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "people", Fingerprint: "source-v1", Version: 1, Fields: []domain.FieldSchema{
		{Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted}, {Name: "name", DataType: "text", Sensitivity: domain.SensitivityConfidential}, {Name: "birthday", DataType: "text"}, {Name: "identity", DataType: "text", Sensitivity: domain.SensitivityRestricted}, {Name: "city", DataType: "text"},
	}}
	target := source
	target.ID = "target"
	target.Schema = "masked"
	target.Fingerprint = "target-v1"
	mappings := []domain.FieldMapping{
		{SourceField: "mobile", TargetField: "mobile", Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "2"}}}},
		{SourceField: "name", TargetField: "name", Strategies: []domain.Strategy{{Kind: domain.StrategyReplace, Parameters: map[string]string{"mode": "constant", "value": "匿名用户"}}}},
		{SourceField: "birthday", TargetField: "birthday", Strategies: []domain.Strategy{{Kind: domain.StrategyGeneralize, Parameters: map[string]string{"mode": "date_year"}}}},
		{SourceField: "identity", TargetField: "identity", Strategies: []domain.Strategy{{Kind: domain.StrategyHash, Parameters: map[string]string{"secret_reference": "hash/default"}}}},
		{SourceField: "city", TargetField: "city", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}},
	}
	plan, conflicts, err := NewCompiler(registry).Compile(source, target, mappings)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("compile err=%v conflicts=%+v", err, conflicts)
	}
	rows, _, counters, err := NewPreviewEngine().Apply(context.Background(), plan, []Row{{"mobile": "13800138000", "name": "张华", "birthday": "1991-07-16T00:00:00Z", "identity": "31010119910716001X", "city": "上海"}})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0]["mobile"] != "138******00" || rows[0]["name"] != "匿名用户" || rows[0]["birthday"] != "1991" || rows[0]["city"] != "上海" {
		t.Fatalf("unexpected output: %+v", rows[0])
	}
	if rows[0]["identity"] == "31010119910716001X" {
		t.Fatal("hash strategy kept plaintext")
	}
	for _, kind := range []string{"mask", "replace", "generalize", "hash", "keep"} {
		if counters[kind] != 1 {
			t.Fatalf("counter %s=%d", kind, counters[kind])
		}
	}
	mappings[4].TargetField = "mobile"
	_, conflicts, err = NewCompiler(registry).Compile(source, target, mappings)
	if err != nil || len(conflicts) == 0 || conflicts[0].Code != "TARGET_COLLISION" {
		t.Fatalf("expected collision, got err=%v conflicts=%+v", err, conflicts)
	}
}

func TestMultipleStrategiesComposeOnTheSameFieldInDeclaredOrder(t *testing.T) {
	registry, err := NewRegistry(MaskTransformer{}, NewHashTransformer(staticSecret{value: []byte("composition-secret")}))
	if err != nil {
		t.Fatal(err)
	}
	table := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "people", Fingerprint: "fp", Version: 1, Fields: []domain.FieldSchema{{Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted}}}
	target := table
	target.ID, target.Schema = "target", "masked"
	mappings := []domain.FieldMapping{{SourceField: "mobile", TargetField: "mobile", Strategies: []domain.Strategy{
		{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "2"}},
		{Kind: domain.StrategyHash, Parameters: map[string]string{"secret_reference": "hash/default", "length": "16"}},
	}}}
	plan, conflicts, err := NewCompiler(registry).Compile(table, target, mappings)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("compile err=%v conflicts=%+v", err, conflicts)
	}
	rows, evidence, counters, err := NewPreviewEngine().Apply(context.Background(), plan, []Row{{"mobile": "13800138000"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows[0]["mobile"].(string)) != 16 || counters["mask"] != 1 || counters["hash"] != 1 {
		t.Fatalf("rows=%+v counters=%+v", rows, counters)
	}
	if len(evidence[0].Changes) != 1 || evidence[0].Changes[0].Rule != "[mask hash]" {
		t.Fatalf("evidence=%+v", evidence)
	}
}

type staticSecret struct{ value []byte }

func (s staticSecret) ResolveSecret(context.Context, string) ([]byte, error) {
	return append([]byte(nil), s.value...), nil
}
