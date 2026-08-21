package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
	"github.com/wyw14/cry052/internal/service"
)

func TestConfirmedPreviewLocksApprovedPolicyScopeStrategiesAndMappings(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	samples := platform.NewSampleDatabase()
	now := time.Now().UTC()
	datasource, _ := domain.NewDataSource("ds", "Local", domain.DataSourceSample, "secret/local", now)
	_ = store.CreateDataSource(ctx, datasource)
	fields := []domain.FieldSchema{{Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted}}
	source := domain.TableSchema{ID: "source", DataSourceID: datasource.ID, Schema: "sample", Name: "people", Fields: fields, Fingerprint: "source-v1", Version: 1}
	target := source
	target.ID, target.Schema, target.Fingerprint = "target", "masked", "target-v1"
	_ = store.PutTable(ctx, source)
	_ = store.PutTable(ctx, target)
	samples.Seed(source.QualifiedName(), []service.Row{{"mobile": "13800138000"}})
	samples.Seed(target.QualifiedName(), nil)
	registry, _ := service.NewRegistry(service.MaskTransformer{}, service.KeepTransformer{})
	previewEngine := service.NewPreviewEngine()
	app := New(Dependencies{Store: store, Samples: samples, Notifier: platform.NewLocalNotifier(time.Now), Redactor: platform.NewRedactor(), Callbacks: platform.NewLocalCallbackSink(), Scheduler: platform.NewLocalScheduler(), Compiler: service.NewCompiler(registry), Preview: previewEngine, Processor: service.NewBatchProcessor(samples, store, previewEngine, 10, time.Now), Exporter: service.NewReportExporter()})
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-policy-lock"}
	mapping := domain.FieldMapping{SourceField: "mobile", TargetField: "mobile", Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "2"}}}}

	wrongScope := domain.PolicyVersion{ID: "wrong-scope", GroupID: "group-a", Name: "Wrong scope", Version: 1, Status: domain.PolicyApproved, Scopes: []string{"sample.other"}, Strategies: []domain.Strategy{{Kind: domain.StrategyMask}}, Revision: 3, CreatedAt: now}
	_ = store.CreatePolicy(ctx, wrongScope)
	if _, err := app.CreatePreview(ctx, actor, CreatePreview{SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: wrongScope.ID, Mappings: []domain.FieldMapping{mapping}, Limit: 5}); err == nil {
		t.Fatal("policy outside source scope was accepted")
	}

	wrongStrategy := wrongScope
	wrongStrategy.ID, wrongStrategy.GroupID, wrongStrategy.Scopes = "wrong-strategy", "group-b", []string{source.QualifiedName()}
	wrongStrategy.Strategies = []domain.Strategy{{Kind: domain.StrategyKeep}}
	_ = store.CreatePolicy(ctx, wrongStrategy)
	if _, err := app.CreatePreview(ctx, actor, CreatePreview{SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: wrongStrategy.ID, Mappings: []domain.FieldMapping{mapping}, Limit: 5}); err == nil {
		t.Fatal("strategy outside approved policy was accepted")
	}

	approved := wrongStrategy
	approved.ID, approved.GroupID, approved.Strategies = "approved", "group-c", []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "2"}}}
	_ = store.CreatePolicy(ctx, approved)
	preview, err := app.CreatePreview(ctx, actor, CreatePreview{SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: approved.ID, Mappings: []domain.FieldMapping{mapping}, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	preview, err = app.ConfirmPreview(ctx, actor, preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := app.CreateBatch(ctx, actor, CreateBatch{PreviewID: preview.ID, IdempotencyKey: "locked-mapping"})
	if err != nil {
		t.Fatal(err)
	}
	tampered := mapping
	tampered.Strategies = []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "0", "suffix": "0"}}}
	if err := app.RunBatch(ctx, actor, batch.ID, []domain.FieldMapping{tampered}); err == nil || !errors.As(err, new(*domain.ValidationError)) {
		t.Fatalf("tampered mapping error=%v", err)
	}
	changedSource := source
	changedSource.Fingerprint = "source-v2"
	changedSource.Version++
	if err := store.SaveTable(ctx, changedSource, source.Version); err != nil {
		t.Fatal(err)
	}
	if err := app.RunBatch(ctx, actor, batch.ID, preview.Mappings); !errors.Is(err, domain.ErrInputSnapshotChanged) {
		t.Fatalf("snapshot change error=%v", err)
	}
	changedVersion := changedSource.Version
	changedSource.Fingerprint = source.Fingerprint
	changedSource.Version++
	if err := store.SaveTable(ctx, changedSource, changedVersion); err != nil {
		t.Fatal(err)
	}
	if err := app.RunBatch(ctx, actor, batch.ID, preview.Mappings); err != nil {
		t.Fatal(err)
	}
}

func TestDiagnosisConflictingPreviewCanBeConfirmedOutsidePolicyBoundary(t *testing.T) {
	TestConfirmedPreviewLocksApprovedPolicyScopeStrategiesAndMappings(t)
}
