package application

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
	"github.com/wyw14/cry052/internal/service"
)

type failingAuditStore struct{ *memory.Store }

func (failingAuditStore) AppendAudit(context.Context, domain.AuditEvent) error {
	return fmt.Errorf("injected audit failure")
}

func auditedBatchFixture(t *testing.T, store Store, stateStore *memory.Store, completed bool) (*App, *platform.SampleDatabase, domain.Batch, []domain.FieldMapping) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC)
	source := domain.TableSchema{ID: "audit-source", DataSourceID: "audit-ds", Schema: "sample", Name: "source", Fields: []domain.FieldSchema{{Name: "id", DataType: "text"}}, Fingerprint: "audit-input-v1", Version: 1}
	target := source
	target.ID = "audit-target"
	target.Schema = "masked"
	target.Name = "target"
	target.Fingerprint = "audit-target-v1"
	if err := stateStore.PutTable(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.PutTable(ctx, target); err != nil {
		t.Fatal(err)
	}
	mappings := []domain.FieldMapping{{SourceField: "id", TargetField: "id", Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}}}
	preview := domain.Preview{ID: "audit-preview", SourceTableID: source.ID, TargetTableID: target.ID, InputFingerprint: source.Fingerprint, Mappings: mappings, StrategyCounters: map[string]int{"keep": 1}, ConfirmedBy: "admin", ConfirmedAt: &now, CreatedAt: now}
	if err := stateStore.CreatePreview(ctx, preview); err != nil {
		t.Fatal(err)
	}
	batch, err := domain.NewBatch("audit-batch", preview, source.QualifiedName(), target.QualifiedName(), "audit-key", "admin", now)
	if err != nil {
		t.Fatal(err)
	}
	if completed {
		batch.Status = domain.BatchCompleted
		batch.RollbackCheckpoint = "0"
		batch.InitialTargetRows = 0
		batch.FinalTargetRows = 1
		batch.Progress = domain.BatchProgress{Read: 1, Written: 1, LastRowID: "1"}
	}
	if err := stateStore.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	samples := platform.NewSampleDatabase()
	samples.Seed(source.QualifiedName(), []service.Row{{"id": "1"}})
	if completed {
		samples.Seed(target.QualifiedName(), []service.Row{{"id": "1"}})
	} else {
		samples.Seed(target.QualifiedName(), nil)
	}
	registry, err := service.NewRegistry(service.KeepTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	previewEngine := service.NewPreviewEngine()
	processor := service.NewBatchProcessor(samples, stateStore, previewEngine, 1, func() time.Time { return now })
	app := New(Dependencies{Store: store, Samples: samples, Redactor: platform.NewRedactor(), Compiler: service.NewCompiler(registry), Preview: previewEngine, Processor: processor, Exporter: service.NewReportExporter(), Now: func() time.Time { return now }, NewID: func() string { return "audit-event" }})
	return app, samples, batch, mappings
}

func TestLocalSideEffectsAreCompensatedWhenAuditPersistenceFails(t *testing.T) {
	root := t.TempDir()
	files, err := platform.NewFileStore(root, 1024, "text/csv")
	if err != nil {
		t.Fatal(err)
	}
	scheduler := platform.NewLocalScheduler()
	app := New(Dependencies{Store: failingAuditStore{memory.New()}, Files: files, Scheduler: scheduler, Redactor: platform.NewRedactor(), NewID: func() string { return "compensated" }})
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-compensation"}
	if _, err := app.SaveAttachment(context.Background(), actor, "export.csv", "text/csv", strings.NewReader("id\n1\n")); err == nil {
		t.Fatal("expected attachment audit failure")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("attachment remained after compensation: %+v", entries)
	}
	if _, err := app.ScheduleLocalEvent(context.Background(), actor, "preview", time.Now().Add(time.Hour), map[string]string{"policy": "p"}); err == nil {
		t.Fatal("expected schedule audit failure")
	}
	events, err := scheduler.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("schedule remained after compensation: %+v", events)
	}
}

func TestRunBatchRefusesTargetWritesWhenAuditIntentFails(t *testing.T) {
	stateStore := memory.New()
	app, samples, batch, mappings := auditedBatchFixture(t, failingAuditStore{stateStore}, stateStore, false)
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-run-intent"}
	if err := app.RunBatch(context.Background(), actor, batch.ID, mappings); err == nil {
		t.Fatal("expected audit intent failure")
	}
	count, err := samples.Count(context.Background(), batch.TargetTable)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := stateStore.GetBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || persisted.Status != domain.BatchPending || persisted.Progress.Written != 0 {
		t.Fatalf("side effect escaped failed intent: count=%d batch=%+v", count, persisted)
	}
}

func TestRollbackBatchRefusesTargetChangesWhenAuditIntentFails(t *testing.T) {
	stateStore := memory.New()
	app, samples, batch, _ := auditedBatchFixture(t, failingAuditStore{stateStore}, stateStore, true)
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-rollback-intent"}
	if _, err := app.RollbackBatch(context.Background(), actor, batch.ID); err == nil {
		t.Fatal("expected audit intent failure")
	}
	count, err := samples.Count(context.Background(), batch.TargetTable)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := stateStore.GetBatch(context.Background(), batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || persisted.Status != domain.BatchCompleted {
		t.Fatalf("rollback escaped failed intent: count=%d batch=%+v", count, persisted)
	}
}

func TestExportReportReturnsNoPayloadWhenAuditIntentFails(t *testing.T) {
	stateStore := memory.New()
	app, _, batch, _ := auditedBatchFixture(t, failingAuditStore{stateStore}, stateStore, true)
	actor := Actor{ID: "auditor", Role: "auditor", RequestID: "request-export-intent"}
	payload, contentType, err := app.ExportReport(context.Background(), actor, batch.ID, "json")
	if err == nil {
		t.Fatal("expected audit intent failure")
	}
	if payload != nil || contentType != "" {
		t.Fatalf("export escaped failed intent: payload=%q contentType=%q", payload, contentType)
	}
}
