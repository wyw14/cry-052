package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
	"github.com/wyw14/cry052/internal/service"
)

func TestOfflinePreviewBatchPublishesGovernanceEvidence(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	samples := platform.NewSampleDatabase()
	callbacks := platform.NewLocalCallbackSink()
	now := time.Now().UTC()
	ds, _ := domain.NewDataSource("ds", "样例", domain.DataSourceSample, "secret/local", now)
	_ = store.CreateDataSource(ctx, ds)
	fields := []domain.FieldSchema{{Name: "id", DataType: "text"}, {Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted}}
	source := domain.TableSchema{ID: "source", DataSourceID: "ds", Schema: "sample", Name: "people", Fields: fields, Fingerprint: "fp", Version: 1}
	target := source
	target.ID = "target"
	target.Schema = "masked"
	target.Fingerprint = "target-fp"
	_ = store.PutTable(ctx, source)
	_ = store.PutTable(ctx, target)
	samples.Seed(source.QualifiedName(), []service.Row{{"id": "1", "mobile": "13800138000"}, {"id": "2", "mobile": "13900139000"}})
	samples.Seed(target.QualifiedName(), nil)
	policy := domain.PolicyVersion{ID: "policy", GroupID: "group", Name: "手机号保护", Version: 1, Status: domain.PolicyApproved, Scopes: []string{source.QualifiedName()}, Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "4"}}, {Kind: domain.StrategyKeep}}, Revision: 3, CreatedAt: now}
	_ = store.CreatePolicy(ctx, policy)
	registry, _ := service.NewRegistry(service.MaskTransformer{}, service.KeepTransformer{})
	previewEngine := service.NewPreviewEngine()
	processor := service.NewBatchProcessor(samples, store, previewEngine, 1, time.Now)
	index := 0
	newID := func() string { index++; return fmt.Sprintf("generated-%d", index) }
	app := New(Dependencies{Store: store, Samples: samples, Notifier: platform.NewLocalNotifier(time.Now), Redactor: platform.NewRedactor(), Callbacks: callbacks, Scheduler: platform.NewLocalScheduler(), Compiler: service.NewCompiler(registry), Preview: previewEngine, Processor: processor, Exporter: service.NewReportExporter(), Now: func() time.Time { return now }, NewID: newID})
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-1"}
	mappings := []domain.FieldMapping{{SourceField: "id", TargetField: "id", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}, {SourceField: "mobile", TargetField: "mobile", Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "4"}}}}}
	preview, err := app.CreatePreview(ctx, actor, CreatePreview{SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: policy.ID, Mappings: mappings, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Rows) != 2 || preview.Rows[0].Changes[0].Before == preview.Rows[0].Changes[0].After {
		t.Fatalf("preview=%+v", preview)
	}
	preview, err = app.ConfirmPreview(ctx, actor, preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := app.CreateBatch(ctx, actor, CreateBatch{PreviewID: preview.ID, IdempotencyKey: "workflow-key"})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.RunBatch(ctx, actor, batch.ID, mappings); err != nil {
		t.Fatal(err)
	}
	report, err := app.ExecutionReport(ctx, actor, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Reconciled() || report.SourceRows != 2 || report.TargetRows != 2 {
		t.Fatalf("report=%+v", report)
	}
	payload, contentType, err := app.ExportReport(ctx, actor, batch.ID, "csv")
	if err != nil || contentType != "text/csv; charset=utf-8" || !strings.Contains(string(payload), "source_rows,2") {
		t.Fatalf("csv=%q contentType=%q err=%v", payload, contentType, err)
	}
	events, err := app.ListLocalCallbacks(ctx, actor, domain.PageRequest{Page: 1, Size: 20, Sort: "created_at"})
	if err != nil || len(events.Items) != 1 || events.Items[0].Topic != "batch.completed" {
		t.Fatalf("callbacks=%+v err=%v", events, err)
	}
	rolledBack, err := app.RollbackBatch(ctx, actor, batch.ID)
	if err != nil || rolledBack.Status != domain.BatchRolledBack {
		t.Fatalf("rollback=%+v err=%v", rolledBack, err)
	}
	audit, err := app.ListAudit(ctx, actor, domain.PageRequest{Page: 1, Size: 50, Sort: "created_at"})
	if err != nil || audit.Total < 4 {
		t.Fatalf("audit=%+v err=%v", audit, err)
	}
	intents := map[string]bool{}
	for _, event := range audit.Items {
		if event.Outcome != "intent" {
			continue
		}
		if event.RequestID != actor.RequestID || event.Metadata["request_id"] != actor.RequestID {
			t.Fatalf("intent lost request correlation: %+v", event)
		}
		intents[event.Action] = true
	}
	for _, action := range []string{"batch.run", "report.export", "batch.rollback"} {
		if !intents[action] {
			t.Fatalf("missing %s intent in %+v", action, audit.Items)
		}
	}
}
