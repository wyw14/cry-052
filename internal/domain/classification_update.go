package domain

import (
	"sort"
	"strings"
)

// replaceClassification applies a partial classification patch to a table.
//
// The patch is sparse: it lists only the fields being reclassified, not a full
// catalog snapshot. Fields absent from the patch are preserved verbatim so a
// one-field edit never drops the rest of the table. Within a patched field,
// only the classification facets supplied in the patch (sensitivity, category,
// scopes) are overwritten; the immutable structural facets (data type,
// nullability) always come from the existing catalog, never the patch. Scope
// lists are normalized (trimmed, de-duplicated, sorted) so duplicate or
// unordered entries never persist as-is.
func replaceClassification(table TableSchema, fields []FieldSchema) (TableSchema, error) {
	if err := validateClassificationPatch(fields); err != nil {
		return TableSchema{}, err
	}

	patchByName := make(map[string]FieldSchema, len(fields))
	for _, field := range fields {
		patchByName[strings.ToLower(strings.TrimSpace(field.Name))] = field
	}

	updated := make([]FieldSchema, 0, len(table.Fields))
	patched := make(map[string]struct{}, len(fields))
	for _, field := range table.Fields {
		key := strings.ToLower(strings.TrimSpace(field.Name))
		patch, ok := patchByName[key]
		if !ok {
			updated = append(updated, field)
			continue
		}
		patched[key] = struct{}{}
		// Start from the existing field so DataType and Nullable (the immutable
		// structural facets) are preserved; only classification facets supplied
		// by the patch are layered on top.
		merged := field
		if patch.Sensitivity != "" {
			merged.Sensitivity = patch.Sensitivity
		}
		if patch.Category != "" {
			merged.Category = patch.Category
		}
		if patch.Scopes != nil {
			merged.Scopes = normalizeScopes(patch.Scopes)
		}
		updated = append(updated, merged)
	}

	if len(patched) != len(patchByName) {
		return TableSchema{}, NewValidationError("classification patch references unknown field", FieldViolation{Field: "fields", Message: "patched field names must match existing catalog fields"})
	}

	table.Fields = updated
	table.Version++
	if err := table.Validate(); err != nil {
		return TableSchema{}, err
	}
	return table, nil
}

func validateClassificationPatch(fields []FieldSchema) error {
	if len(fields) == 0 {
		return NewValidationError("classification patch is empty", FieldViolation{Field: "fields", Message: "at least one field is required"})
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if name == "" {
			return NewValidationError("classification field is missing a name", FieldViolation{Field: "fields", Message: "field name is required"})
		}
		if field.Sensitivity == "" && field.Category == "" && field.Scopes == nil {
			return NewValidationError("classification field provides no changes", FieldViolation{Field: "fields", Message: field.Name + " must specify sensitivity, category, or scopes"})
		}
		if _, exists := seen[name]; exists {
			return NewValidationError("duplicate field in classification patch", FieldViolation{Field: "fields", Message: field.Name + " appears more than once"})
		}
		seen[name] = struct{}{}
	}
	return nil
}

// normalizeScopes trims, de-duplicates, and sorts scope entries so the same
// input always yields the same stored list. An empty input yields an empty
// (non-nil) slice rather than nil.
func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(scopes))
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}
