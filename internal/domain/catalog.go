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

func (t *TableSchema) ApplyClassification(fields []FieldSchema) error {
	updated, err := replaceClassification(*t, fields)
	if err != nil {
		return err
	}
	*t = updated
	return nil
}
