package domain

import (
	"strings"
	"time"
)

const (
	AuditIntent  = "intent"
	AuditSuccess = "success"
	AuditFailed  = "failed"
)

type AuditEvent struct {
	CreatedAt     time.Time         `json:"created_at"`
	ID            string            `json:"id"`
	Outcome       string            `json:"outcome"`
	RequestID     string            `json:"request_id"`
	Metadata      map[string]string `json:"metadata"`
	Resource      string            `json:"resource"`
	Action        string            `json:"action"`
	Actor         string            `json:"actor"`
	Category      string            `json:"category"`
	SchemaVersion int               `json:"schema_version"`
}

func NewAuditEvent(id, requestID, actor, action, resource, outcome string, metadata map[string]string, createdAt time.Time) AuditEvent {
	return AuditEvent{
		ID:            id,
		RequestID:     requestID,
		Actor:         actor,
		Action:        action,
		Resource:      resource,
		Outcome:       outcome,
		Metadata:      cloneAuditMetadata(metadata),
		CreatedAt:     createdAt,
		Category:      auditCategory(action),
		SchemaVersion: 1,
	}
}

func auditCategory(action string) string {
	category, _, found := strings.Cut(action, ".")
	if !found || category == "" {
		return "governance"
	}
	return category
}

func cloneAuditMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
