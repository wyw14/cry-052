package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/postgres"
)

func TestSeedDemoIsIdempotentAndLoadsSampleRows(t *testing.T) {
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	samples := platform.NewSampleDatabase()
	if err := seedDemo(ctx, store, samples); err != nil {
		t.Fatal(err)
	}
	if err := seedDemo(ctx, store, samples); err != nil {
		t.Fatal(err)
	}
	tables, err := store.ListTables(ctx, "demo-source", domain.PageRequest{Page: 1, Size: 10, Sort: "qualified_name"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tables.Items) != 2 {
		t.Fatalf("tables=%+v", tables)
	}
	var sourceIndex = -1
	for index, table := range tables.Items {
		if table.ID == "demo-customers" {
			sourceIndex = index
		}
	}
	if sourceIndex < 0 {
		t.Fatal("demo source table missing")
	}
	rows, err := samples.Rows(ctx, tables.Items[sourceIndex], 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0]["mobile"] == "" {
		t.Fatalf("rows=%+v", rows)
	}
}
