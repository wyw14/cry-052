package domain

import (
	"regexp"
	"strings"
)

// inlineConnectionMaterial matches the connection primitives that auditors do
// not want persisted: DSN schemes (anything containing "://") and libpq
// key=value pairs (password=..., user=, host=..., dbname=...). It mirrors the
// signals the platform redactor already treats as secret-bearing so the
// registration gate and the log scrubber agree on what counts as inline
// material.
var inlineConnectionMaterial = regexp.MustCompile(`(?i)(://|\b(?:host|hostname|port|user|username|password|passwd|pwd|dbname|database|sslmode|token|secret|dsn)\s*[=:]\s*\S+)`)

func normalizeConnectionReference(raw string) (string, error) {
	reference := strings.TrimSpace(raw)
	if reference == "" {
		return "", NewValidationError("data source is invalid", FieldViolation{Field: "connection_reference", Message: "use an encrypted local reference, not a DSN"})
	}
	// Local references resolve to a secret held by the host (for example
	// "secret/demo" or "hash/default"). They never carry inline connection
	// material, so reject anything that looks like a DSN or embedded
	// credential before it can be persisted in cleartext.
	if looksLikeInlineConnectionMaterial(reference) {
		return "", NewValidationError("data source is invalid", FieldViolation{Field: "connection_reference", Message: "use an encrypted local reference, not a DSN"})
	}
	// Older deployments used several provider-specific aliases. Unknown text is
	// preserved here so those aliases can still be resolved by the local host.
	return reference, nil
}

func looksLikeInlineConnectionMaterial(reference string) bool {
	if inlineConnectionMaterial.MatchString(reference) {
		return true
	}
	// "user:pass@host" and "user@host" forms embed credentials inline. Such
	// userinfo/host separators never appear in a local secret reference, so
	// their presence is treated as connection material. This is checked
	// directly because a leading "user:" segment is parsed as a URI scheme,
	// which would otherwise hide the embedded credentials.
	return strings.Contains(reference, "@")
}
