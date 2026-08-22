package domain

import "testing"

func TestPhysicalIdentityFoldsCaseWhitespaceAndQuotes(t *testing.T) {
	source := TableSchema{ID: "catalog-a", DataSourceID: "local-primary", Schema: "Public", Name: "Customers"}
	cases := []struct {
		name   string
		target TableSchema
	}{
		{"case folded", TableSchema{ID: "catalog-b", DataSourceID: "local-primary", Schema: "public", Name: "customers"}},
		{"surrounding whitespace", TableSchema{ID: "catalog-c", DataSourceID: " LOCAL-PRIMARY ", Schema: " public ", Name: " customers "}},
		{"double-quoted alias", TableSchema{ID: "catalog-d", DataSourceID: "local-primary", Schema: `"public"`, Name: `"customers"`}},
	}
	for _, tc := range cases {
		if !source.PhysicalIdentity().Same(tc.target.PhysicalIdentity()) {
			t.Errorf("%q alias %q was treated as a distinct physical table", tc.name, tc.target.QualifiedName())
		}
	}
}

func TestPhysicalIdentityKeepsSeparateTargetDistinct(t *testing.T) {
	source := TableSchema{ID: "catalog-a", DataSourceID: "local-primary", Schema: "public", Name: "customers"}
	separate := TableSchema{ID: "catalog-z", DataSourceID: "local-primary", Schema: "masked", Name: "customers"}
	if source.PhysicalIdentity().Same(separate.PhysicalIdentity()) {
		t.Fatal("independent masked target was collapsed onto its source")
	}
}
