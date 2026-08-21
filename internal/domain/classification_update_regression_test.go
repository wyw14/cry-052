package domain

import (
	"reflect"
	"testing"
)

func TestSparseClassificationUpdatePreservesCatalogShape(t *testing.T) {
	table := TableSchema{
		ID: "people", DataSourceID: "source", Schema: "public", Name: "people", Version: 4,
		Fields: []FieldSchema{
			{Name: "id", DataType: "uuid", Sensitivity: SensitivityInternal, Category: "identity", Scopes: []string{"ops"}},
			{Name: "email", DataType: "text", Nullable: true, Sensitivity: SensitivityConfidential, Category: "contact", Scopes: []string{"support"}},
		},
	}
	patch := []FieldSchema{{Name: "email", Sensitivity: SensitivityRestricted, Category: "direct_contact", Scopes: []string{"masked_export", "support", "masked_export"}}}
	if err := table.ApplyClassification(patch); err != nil {
		t.Fatalf("apply sparse classification: %v", err)
	}
	if len(table.Fields) != 2 {
		t.Fatalf("sparse edit dropped catalog fields: %#v", table.Fields)
	}
	email, ok := table.Field("email")
	if !ok || email.DataType != "text" || !email.Nullable {
		t.Fatalf("immutable field shape changed: %#v", email)
	}
	if email.Sensitivity != SensitivityRestricted || email.Category != "direct_contact" {
		t.Fatalf("classification was not applied: %#v", email)
	}
	if !reflect.DeepEqual(email.Scopes, []string{"masked_export", "support"}) {
		t.Fatalf("scopes were not normalized: %#v", email.Scopes)
	}
	if table.Version != 5 {
		t.Fatalf("expected one catalog revision, got %d", table.Version)
	}
}
