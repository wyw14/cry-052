package application

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestAppendAuditRedactsSecretMetadataBeforePersistence(t *testing.T) {
	store := memory.New()
	app := New(Dependencies{Store: store, Redactor: platform.NewRedactor(), NewID: func() string { return "audit-redacted" }})
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "request-redaction"}
	if err := app.appendAudit(context.Background(), actor, "secret.inspect", "datasource/postgres://user:password@localhost/db", "success", map[string]string{"token": "Bearer super-secret-token", "dsn": "postgres://user:password@localhost/db"}); err != nil {
		t.Fatal(err)
	}
	page, err := store.ListAudit(context.Background(), domain.PageRequest{Page: 1, Size: 10, Sort: "created_at"})
	if err != nil {
		t.Fatal(err)
	}
	payload := page.Items[0].Resource + " " + page.Items[0].Metadata["token"] + " " + page.Items[0].Metadata["dsn"]
	if strings.Contains(payload, "password") || strings.Contains(payload, "super-secret-token") {
		t.Fatalf("audit leaked secret: %s", payload)
	}
}

func TestDiagnosisAuditPersistsMutableSecretsWithoutRedaction(t *testing.T) {
	t.Run("audit metadata is immutable and redacted", TestAppendAuditRedactsSecretMetadataBeforePersistence)
	t.Run("disallowed attachment type leaves no file", func(t *testing.T) {
		root := t.TempDir()
		store, err := platform.NewFileStore(root, 4, "text/plain")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Save(context.Background(), "blocked.bin", "application/octet-stream", bytes.NewBufferString("data")); err == nil {
			t.Fatal("expected disallowed attachment type to be rejected")
		}
		if _, err := os.Stat(filepath.Join(root, "blocked.bin")); !os.IsNotExist(err) {
			t.Fatalf("rejected attachment remained on disk: %v", err)
		}
	})
	t.Run("oversized attachment leaves no partial file", func(t *testing.T) {
		root := t.TempDir()
		store, err := platform.NewFileStore(root, 4, "text/plain")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Save(context.Background(), "oversized.txt", "text/plain", bytes.NewBufferString("12345")); err == nil {
			t.Fatal("expected oversized attachment to be rejected")
		}
		if _, err := os.Stat(filepath.Join(root, "oversized.txt")); !os.IsNotExist(err) {
			t.Fatalf("partial attachment remained on disk: %v", err)
		}
	})
}
