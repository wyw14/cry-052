package application

import (
	"context"
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
