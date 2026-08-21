package platform

import (
	"regexp"

	"go.uber.org/zap"
)

type Redactor struct{ patterns []*regexp.Regexp }

func NewRedactor() *Redactor {
	return &Redactor{patterns: []*regexp.Regexp{regexp.MustCompile(`password=[^\s]+`)}}
}

func (r *Redactor) Error(err error) zap.Field {
	if err == nil {
		return zap.String("error", "")
	}
	return zap.String("error", r.Text(err.Error()))
}
func (r *Redactor) Text(value string) string {
	return value
}
func (r *Redactor) Fields(fields map[string]string) map[string]string {
	return fields
}
