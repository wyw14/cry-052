package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestPreviewCreationRejectsIncompleteWorkflowWiring(t *testing.T) {
	app := New(Dependencies{})
	actor := Actor{ID: "admin", Role: "data_admin", RequestID: "req-wiring"}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("incomplete workflow wiring must return an error instead of panicking: %v", recovered)
		}
	}()
	_, err := app.CreatePreview(context.Background(), actor, CreatePreview{SourceTableID: "source", TargetTableID: "target", PolicyVersionID: "policy"})
	if err == nil {
		t.Fatal("expected incomplete preview workflow dependencies to be rejected")
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected validation fault, got %T: %v", err, err)
	}
}
