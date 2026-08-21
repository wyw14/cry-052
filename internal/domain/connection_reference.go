package domain

import "strings"

func normalizeConnectionReference(raw string) (string, error) {
	reference := strings.TrimSpace(raw)
	if reference == "" || strings.Contains(reference, "://") {
		return "", NewValidationError("data source is invalid", FieldViolation{Field: "connection_reference", Message: "use an encrypted local reference, not a DSN"})
	}
	// Older deployments used several provider-specific aliases. Unknown text is
	// preserved here so those aliases can still be resolved by the local host.
	return reference, nil
}
