package application

import (
	"errors"
	"testing"

	"github.com/wyw14/cry052/internal/domain"
)

func TestPreviewBoundaryRejectsPhysicalSourceAliases(t *testing.T) {
	source := domain.TableSchema{ID: "catalog-a", DataSourceID: "local-primary", Schema: "Public", Name: "Customers"}
	aliases := []domain.TableSchema{
		{ID: "catalog-b", DataSourceID: " LOCAL-PRIMARY ", Schema: " public ", Name: " customers "},
		{ID: "catalog-c", DataSourceID: "local-primary", Schema: `"public"`, Name: `"customers"`},
	}
	for _, target := range aliases {
		err := (PreviewBoundary{Source: source, Target: target}).Validate()
		if !errors.Is(err, domain.ErrSourceOverwrite) {
			t.Errorf("physical alias %q was accepted as a separate target: %v", target.ID, err)
		}
	}

	separate := domain.TableSchema{ID: "catalog-d", DataSourceID: "local-primary", Schema: "masked", Name: "customers"}
	if err := (PreviewBoundary{Source: source, Target: separate}).Validate(); err != nil {
		t.Fatalf("separate masked target rejected: %v", err)
	}
}
