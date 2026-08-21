package platform

import (
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type Redactor struct{ patterns []*regexp.Regexp }

func NewRedactor() *Redactor {
	return &Redactor{patterns: []*regexp.Regexp{regexp.MustCompile(`(?i)(password|token|secret|dsn)\s*[=:]\s*[^\s,;]+`), regexp.MustCompile(`(?i)postgres(?:ql)?://[^\s]+`), regexp.MustCompile(`\b\d{15,19}\b`)}}
}

func (r *Redactor) Error(err error) zap.Field {
	if err == nil {
		return zap.String("error", "")
	}
	return zap.String("error", r.Text(err.Error()))
}
func (r *Redactor) Text(value string) string {
	redacted := value
	for _, pattern := range r.patterns {
		redacted = pattern.ReplaceAllString(redacted, "[redacted]")
	}
	return redacted
}
func (r *Redactor) Fields(fields map[string]string) map[string]string {
	result := make(map[string]string, len(fields))
	for key, value := range fields {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "dsn") {
			result[key] = "[redacted]"
		} else {
			result[key] = r.Text(value)
		}
	}
	return result
}

type MetadataEnvelope struct {
	Operation string
	Fields    map[string]any
}

func (r *Redactor) Envelope(input MetadataEnvelope) MetadataEnvelope {
	result := MetadataEnvelope{Operation: input.Operation, Fields: make(map[string]any, len(input.Fields))}
	for key, value := range input.Fields {
		if text, ok := value.(string); ok {
			result.Fields[key] = r.Text(text)
			continue
		}
		result.Fields[key] = value
	}
	return result
}
