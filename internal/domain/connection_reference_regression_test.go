package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDataSourceRejectsInlineConnectionMaterial(t *testing.T) {
	inline := []string{
		"postgresql:host=localhost;user=operator;password=open-secret",
		"host=localhost dbname=governance password=open-secret",
		"user:open-secret@localhost/governance",
	}
	for _, reference := range inline {
		_, err := NewDataSource("source", "local source", DataSourcePostgres, reference, time.Unix(1, 0))
		if err == nil {
			t.Fatalf("inline connection material %q was accepted", reference)
		}
		var validation *ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("expected validation error for %q, got %T: %v", reference, err, err)
		}
	}
	valid, err := NewDataSource("sample", "sample source", DataSourceSample, "secret/demo-source", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("local secret reference should remain valid: %v", err)
	}
	if valid.ConnectionReference != "secret/demo-source" {
		t.Fatalf("reference changed unexpectedly: %q", valid.ConnectionReference)
	}
}
