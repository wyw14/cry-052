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

func TestCreateBatchReplaysOnlyEquivalentIdempotencyInput(t *testing.T) {
	store := memory.New()
	samples := platform.NewSampleDatabase()
	previewEngine := service.NewPreviewEngine()
	processor := service.NewBatchProcessor(samples, store, previewEngine, 10, time.Now)
	registry, _ := service.NewRegistry(service.KeepTransformer{})
	app := New(Dependencies{Store: store, Samples: samples, Notifier: platform.NewLocalNotifier(time.Now), Compiler: service.NewCompiler(registry), Preview: previewEngine, Processor: processor, Exporter: service.NewReportExporter(), NewID: func() string { return "batch-1" }})
	now := time.Now()
	source := domain.TableSchema{ID: "s", DataSourceID: "d", Schema: "public", Name: "people", Fields: []domain.FieldSchema{{Name: "id", DataType: "text"}}, Fingerprint: "fp", Version: 1}
	target := source
	target.ID = "t"
	target.Schema = "masked"
	target.Fingerprint = "target"
	ds, _ := domain.NewDataSource("d", "sample", domain.DataSourceSample, "secret/demo", now)
	_ = store.CreateDataSource(context.Background(), ds)
	_ = store.PutTable(context.Background(), source)
	_ = store.PutTable(context.Background(), target)
	preview := domain.Preview{ID: "p", SourceTableID: "s", TargetTableID: "t", InputFingerprint: "fp", ConfirmedBy: "admin", ConfirmedAt: &now}
	_ = store.CreatePreview(context.Background(), preview)
	actor := Actor{ID: "admin", Role: "data_admin"}
	first, err := app.CreateBatch(context.Background(), actor, CreateBatch{PreviewID: "p", IdempotencyKey: "same"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.CreateBatch(context.Background(), actor, CreateBatch{PreviewID: "p", IdempotencyKey: "same"})
	if err != nil || second.ID != first.ID {
		t.Fatalf("idempotent replay failed: %+v %v", second, err)
	}
	preview.ID = "other"
	_ = store.CreatePreview(context.Background(), preview)
	_, err = app.CreateBatch(context.Background(), actor, CreateBatch{PreviewID: "other", IdempotencyKey: "same"})
	if !errors.Is(err, domain.ErrIdempotencyReuse) {
		t.Fatalf("expected key reuse rejection, got %v", err)
	}
}
