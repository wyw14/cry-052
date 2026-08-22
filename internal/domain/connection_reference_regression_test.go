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
		// URL-style DSNs with embedded credentials.
		"postgres://operator:open-secret@localhost:5432/governance",
		"postgresql://operator:open-secret@localhost/governance?sslmode=disable",
		// libpq key=value material without a scheme.
		"host=db.local user=operator password=open-secret dbname=governance",
		// Provider schemes that imply a live connection string.
		"mysql://operator:open-secret@localhost/governance",
		"redis://:open-secret@localhost:6379/0",
		// Credential-bearing fragments on their own.
		"password=open-secret",
		"user=operator password=open-secret",
		"dsn=postgres://operator:open-secret@localhost/db",
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

func TestDataSourceAcceptsLocalSecretReferences(t *testing.T) {
	// References that resolve to a secret held by the host (never inline
	// connection material) must remain valid for both data source kinds.
	references := []string{
		"secret/demo",
		"secret/demo-source",
		"hash/default",
		"vault/prod/governance-readonly",
	}
	for _, reference := range references {
		source, err := NewDataSource("ds-"+reference, "local source", DataSourcePostgres, reference, time.Unix(1, 0))
		if err != nil {
			t.Fatalf("local reference %q should be valid: %v", reference, err)
		}
		if source.ConnectionReference != reference {
			t.Fatalf("reference %q was altered to %q", reference, source.ConnectionReference)
		}
	}
}

func TestDataSourceRejectsBlankConnectionReference(t *testing.T) {
	if _, err := NewDataSource("ds", "local source", DataSourceSample, "   ", time.Unix(1, 0)); err == nil {
		t.Fatal("blank connection reference was accepted")
	}
}
