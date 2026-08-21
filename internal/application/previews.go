package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

type CreatePreview struct {
	SourceTableID, TargetTableID, PolicyVersionID string
	Mappings                                      []domain.FieldMapping
	Limit                                         int
}

func (a *App) CreatePreview(ctx context.Context, actor Actor, command CreatePreview) (domain.Preview, error) {
	if err := a.wiring.requirePreviewCreation(); err != nil {
		return domain.Preview{}, err
	}
	if err := actor.Require("data_admin", "policy_editor"); err != nil {
		return domain.Preview{}, err
	}
	source, err := a.store.GetTable(ctx, command.SourceTableID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("source table: %w", err)
	}
	target, err := a.store.GetTable(ctx, command.TargetTableID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("target table: %w", err)
	}
	if source.ID == target.ID || source.QualifiedName() == target.QualifiedName() {
		return domain.Preview{}, domain.ErrSourceOverwrite
	}
	policy, err := a.store.GetPolicy(ctx, command.PolicyVersionID)
	if err != nil {
		return domain.Preview{}, fmt.Errorf("policy: %w", err)
	}
	if policy.Status != domain.PolicyApproved {
		return domain.Preview{}, domain.ErrInvalidTransition
	}
	if !policy.AppliesTo(source.QualifiedName()) {
		return domain.Preview{}, domain.NewValidationError("policy scope does not include the source table")
	}
	for _, mapping := range command.Mappings {
		for _, strategy := range mapping.Strategies {
			if !policy.AllowsStrategy(strategy) {
				return domain.Preview{}, domain.NewValidationError("mapping uses strategy parameters outside the approved policy")
			}
		}
	}
	plan, conflicts, err := a.compiler.Compile(source, target, command.Mappings)
	if err != nil {
		return domain.Preview{}, err
	}
	preview := domain.Preview{ID: a.newID(), SourceTableID: source.ID, TargetTableID: target.ID, PolicyVersionID: policy.ID, InputFingerprint: source.Fingerprint, Mappings: command.Mappings, Conflicts: conflicts, CreatedAt: a.now()}
	if len(conflicts) == 0 {
		rows, err := a.samples.Rows(ctx, source, min(max(command.Limit, 1), 100))
		if err != nil {
			return domain.Preview{}, fmt.Errorf("load local sample rows: %w", err)
		}
		_, preview.Rows, preview.StrategyCounters, err = a.preview.Apply(ctx, plan, rows)
		if err != nil {
			return domain.Preview{}, err
		}
	}
	audit := a.newAuditEvent(actor, "preview.create", "preview/"+preview.ID, "success", map[string]string{"source_table": source.QualifiedName(), "target_table": target.QualifiedName()})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{CreatePreview: &preview, Audit: audit}); err != nil {
		return domain.Preview{}, err
	}
	return preview, nil
}

func (a *App) ConfirmPreview(ctx context.Context, actor Actor, id string) (domain.Preview, error) {
	if err := actor.Require("data_admin"); err != nil {
		return domain.Preview{}, err
	}
	preview, err := a.store.GetPreview(ctx, id)
	if err != nil {
		return domain.Preview{}, err
	}
	if err := preview.Confirm(actor.ID, a.now()); err != nil {
		return domain.Preview{}, err
	}
	audit := a.newAuditEvent(actor, "preview.confirm", "preview/"+id, "success", nil)
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{UpdatePreview: &preview, Audit: audit}); err != nil {
		return domain.Preview{}, err
	}
	return preview, nil
}
