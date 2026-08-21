package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func TestPostgresCoreWorkflowPersistsAcrossConnections(t *testing.T) {
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().Format("150405.000000000")
	now := time.Now().UTC()
	datasource, err := domain.NewDataSource("workflow-ds-"+suffix, "Workflow "+suffix, domain.DataSourceSample, "secret/workflow", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDataSource(ctx, datasource); err != nil {
		t.Fatal(err)
	}
	fields := []domain.FieldSchema{{Name: "id", DataType: "text", Sensitivity: domain.SensitivityInternal, Category: "identifier", Scopes: []string{"governance"}}}
	source := domain.TableSchema{ID: "workflow-source-" + suffix, DataSourceID: datasource.ID, Schema: "sample", Name: "source_" + suffix, Fields: fields, Fingerprint: "source-fp-" + suffix, Version: 1, DiscoveredAt: now}
	target := source
	target.ID = "workflow-target-" + suffix
	target.Name = "target_" + suffix
	target.Fingerprint = "target-fp-" + suffix
	if err := store.PutTable(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := store.PutTable(ctx, target); err != nil {
		t.Fatal(err)
	}
	policy := domain.PolicyVersion{ID: "workflow-policy-" + suffix, GroupID: "workflow-group-" + suffix, Name: "Workflow policy", Version: 1, Status: domain.PolicyApproved, Scopes: []string{source.QualifiedName()}, Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}, Revision: 3, CreatedAt: now}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	preview := domain.Preview{ID: "workflow-preview-" + suffix, SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: policy.ID, InputFingerprint: source.Fingerprint, CreatedAt: now}
	if err := store.CreatePreview(ctx, preview); err != nil {
		t.Fatal(err)
	}
	batch := domain.Batch{ID: "workflow-batch-" + suffix, PreviewID: preview.ID, SourceTable: source.QualifiedName(), TargetTable: target.QualifiedName(), InputFingerprint: source.Fingerprint, IdempotencyKey: "workflow-key-" + suffix, Status: domain.BatchPending, Version: 1, CreatedBy: "admin", CreatedAt: now}
	if err := store.CreateBatch(ctx, batch); err != nil {
		t.Fatal(err)
	}
	store.Close()

	reopened, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	persisted, err := reopened.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.PreviewID != preview.ID || persisted.InputFingerprint != source.Fingerprint {
		t.Fatalf("persisted batch=%+v", persisted)
	}
	page, err := reopened.ListTables(ctx, datasource.ID, domain.PageRequest{Page: 1, Size: 10, Sort: "qualified_name", Filters: map[string]string{"category": "identifier"}})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("table page=%+v", page)
	}
}
