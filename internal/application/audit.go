package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

type Actor struct{ ID, Role, RequestID string }

func (a Actor) Require(roles ...string) error {
	if a.ID == "" {
		return domain.ErrPermissionDenied
	}
	for _, role := range roles {
		if a.Role == role {
			return nil
		}
	}
	return domain.ErrPermissionDenied
}

func (a *App) appendAudit(ctx context.Context, actor Actor, action, resource, outcome string, metadata map[string]string) error {
	event := a.newAuditEvent(actor, action, resource, outcome, metadata)
	if err := a.store.AppendAudit(ctx, event); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}
	return nil
}

func (a *App) appendAuditIntent(ctx context.Context, actor Actor, action, resource string, metadata map[string]string) error {
	if actor.RequestID == "" {
		return domain.NewValidationError("request id is required for audited side effects")
	}
	intentMetadata := make(map[string]string, len(metadata)+1)
	for key, value := range metadata {
		intentMetadata[key] = value
	}
	intentMetadata["request_id"] = actor.RequestID
	return a.appendAudit(ctx, actor, action, resource, domain.AuditIntent, intentMetadata)
}

func (a *App) newAuditEvent(actor Actor, action, resource, outcome string, metadata map[string]string) domain.AuditEvent {
	if a.redactor != nil {
		metadata = a.redactor.Fields(metadata)
		resource = a.redactor.Text(resource)
	}
	return domain.NewAuditEvent(a.newID(), actor.RequestID, actor.ID, action, resource, outcome, metadata, a.now())
}

func (a *App) ListAudit(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.AuditEvent], error) {
	if err := actor.Require("data_admin", "auditor"); err != nil {
		return domain.Page[domain.AuditEvent]{}, err
	}
	request, err := request.Normalize(map[string]struct{}{"created_at": {}, "action": {}}, map[string]struct{}{"actor": {}, "action": {}, "resource": {}, "outcome": {}})
	if err != nil {
		return domain.Page[domain.AuditEvent]{}, err
	}
	return a.store.ListAudit(ctx, request)
}
