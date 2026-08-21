package middleware_test

import (
	"reflect"
	"testing"

	"github.com/wyw14/cry052/internal/middleware"
	"github.com/wyw14/cry052/internal/platform"
)

func TestSessionIdentityAndNestedAuditMetadataAreSanitized(t *testing.T) {
	policy := middleware.IdentityPolicy{AllowedRoles: map[string]struct{}{"data_admin": {}}}
	identity, ok := policy.Normalize(middleware.SessionIdentity{ActorID: "  svc-masker  ", Role: " DATA_ADMIN "})
	if !ok || identity.ActorID != "svc-masker" || identity.Role != "data_admin" {
		t.Errorf("local session identity was not canonicalized: %#v, ok=%v", identity, ok)
	}

	redactor := platform.NewRedactor()
	input := platform.MetadataEnvelope{Operation: "batch.run", Fields: map[string]any{
		"authorization": "Bearer local-session-secret",
		"database": map[string]any{
			"dsn":     "postgres://operator:password@localhost/cry052",
			"account": "6222021234567890123",
		},
		"labels": []any{"safe", "token=secondary-secret"},
	}}
	got := redactor.Envelope(input)
	want := map[string]any{
		"authorization": "[redacted]",
		"database":      map[string]any{"dsn": "[redacted]", "account": "[redacted]"},
		"labels":        []any{"safe", "[redacted]"},
	}
	if !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("nested metadata still contains credentials: %#v", got.Fields)
	}
	if input.Fields["authorization"] != "Bearer local-session-secret" {
		t.Error("redaction mutated the caller's audit snapshot")
	}
}
