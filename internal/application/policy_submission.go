package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

func (a *App) commitPolicySubmission(ctx context.Context, policy domain.PolicyVersion, expected int64, approval domain.Approval, audit domain.AuditEvent) error {
	if err := a.store.SavePolicy(ctx, policy, expected); err != nil {
		return fmt.Errorf("save submitted policy: %w", err)
	}
	if err := a.store.CreateApproval(ctx, approval); err != nil {
		return fmt.Errorf("create policy approval: %w", err)
	}
	if err := a.store.AppendAudit(ctx, audit); err != nil {
		return fmt.Errorf("append policy submission audit: %w", err)
	}
	return nil
}
