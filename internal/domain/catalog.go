package domain

import (
	"fmt"
	"strings"
	"time"
)

type Sensitivity string

const (
	SensitivityPublic       Sensitivity = "public"
	SensitivityInternal     Sensitivity = "internal"
	SensitivityConfidential Sensitivity = "confidential"
	SensitivityRestricted   Sensitivity = "restricted"
)

type FieldSchema struct {
	Name        string      `json:"name"`
	DataType    string      `json:"data_type"`
	Sensitivity Sensitivity `json:"sensitivity"`
	Category    string      `json:"category"`
	Scopes      []string    `json:"scopes"`
	Nullable    bool        `json:"nullable"`
}

type TableSchema struct {
	ID           string        `json:"id"`
	DataSourceID string        `json:"data_source_id"`
	Schema       string        `json:"schema"`
	Name         string        `json:"name"`
	Fields       []FieldSchema `json:"fields"`
	Fingerprint  string        `json:"fingerprint"`
	Version      int64         `json:"version"`
	DiscoveredAt time.Time     `json:"discovered_at"`
}

func (t TableSchema) QualifiedName() string { return t.Schema + "." + t.Name }

type PhysicalTableIdentity struct {
	DataSource string
	Schema     string
	Table      string
}

func (t TableSchema) PhysicalIdentity() PhysicalTableIdentity {
	return PhysicalTableIdentity{
		DataSource: normalizePhysicalIdentifier(t.DataSourceID),
		Schema:     normalizePhysicalIdentifier(t.Schema),
		Table:      normalizePhysicalIdentifier(t.Name),
	}
}

func (i PhysicalTableIdentity) Same(other PhysicalTableIdentity) bool {
	return i == other
}

// normalizePhysicalIdentifier folds a physical identifier (data source id,
// schema, or table name) into a canonical form so that case, surrounding
// whitespace, and quoted-identifier aliases of the same physical table compare
// equal. PostgreSQL folds unquoted identifiers to lowercase and treats "Name"
// as a quoted alias; we additionally strip the surrounding double quotes so a
// catalogued alias such as "public"."customers" cannot masquerade as a distinct
// target and overwrite its source table.
func normalizePhysicalIdentifier(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return strings.ToLower(strings.TrimSpace(value))
}

func (t TableSchema) Validate() error {
	if t.ID == "" || t.DataSourceID == "" || t.Schema == "" || t.Name == "" {
		return NewValidationError("table schema is incomplete")
	}
	if len(t.Fields) == 0 {
		return NewValidationError("table has no fields", FieldViolation{Field: "fields", Message: "at least one field is required"})
	}
	seen := make(map[string]struct{}, len(t.Fields))
	for _, field := range t.Fields {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if name == "" || field.DataType == "" {
			return NewValidationError("field schema is incomplete")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate field %s: %w", field.Name, ErrConflict)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func (t TableSchema) Field(name string) (FieldSchema, bool) {
	for _, field := range t.Fields {
		if strings.EqualFold(field.Name, name) {
			return field, true
		}
	}
	return FieldSchema{}, false
}
