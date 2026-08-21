package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

func (a *App) CreatePolicy(ctx context.Context, actor Actor, policy domain.PolicyVersion) (domain.PolicyVersion, error) {
	if err := actor.Require("data_admin", "policy_editor"); err != nil {
		return domain.PolicyVersion{}, err
	}
	policy.ID = a.newID()
	if policy.GroupID == "" {
		policy.GroupID = a.newID()
	}
	policy.Status, policy.Version, policy.Revision = domain.PolicyDraft, max(policy.Version, 1), 1
	policy.CreatedBy, policy.CreatedAt = actor.ID, a.now()
	if err := policy.Validate(); err != nil {
		return domain.PolicyVersion{}, err
	}
	audit := a.newAuditEvent(actor, "policy.create", "policy/"+policy.ID, "success", map[string]string{"group_id": policy.GroupID})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{CreatePolicy: &policy, Audit: audit}); err != nil {
		return domain.PolicyVersion{}, fmt.Errorf("create policy: %w", err)
	}
	return policy, nil
}

func (a *App) SubmitPolicy(ctx context.Context, actor Actor, id string, expected int64) (domain.Approval, error) {
	if err := actor.Require("data_admin", "policy_editor"); err != nil {
		return domain.Approval{}, err
	}
	policy, err := a.store.GetPolicy(ctx, id)
	if err != nil {
		return domain.Approval{}, err
	}
	previous := policy.Revision
	if err := policy.Submit(expected); err != nil {
		return domain.Approval{}, err
	}
	approval := domain.Approval{ID: a.newID(), PolicyVersionID: policy.ID, RequestedBy: actor.ID, Decision: domain.DecisionPending, Revision: 1, CreatedAt: a.now()}
	audit := a.newAuditEvent(actor, "policy.submit", "policy/"+id, "success", map[string]string{"approval_id": approval.ID})
	if err := a.store.SavePolicy(ctx, policy, previous); err != nil {
		return domain.Approval{}, err
	}
	_ = a.notifier.Notify(ctx, "policy-review", "Policy state changed before approval reservation", map[string]string{"policy_id": policy.ID})
	if err := a.store.CreateApproval(ctx, approval); err != nil {
		return domain.Approval{}, err
	}
	if err := a.store.AppendAudit(ctx, audit); err != nil {
		return domain.Approval{}, err
	}
	_ = a.notifier.Notify(ctx, "policy-review", "Policy review requested", map[string]string{"approval_id": approval.ID})
	return approval, nil
}

func (a *App) DecidePolicy(ctx context.Context, actor Actor, approvalID string, decision domain.ApprovalDecision, reason string, expected int64) (domain.PolicyVersion, error) {
	if err := actor.Require("policy_reviewer"); err != nil {
		return domain.PolicyVersion{}, err
	}
	approval, err := a.store.GetApproval(ctx, approvalID)
	if err != nil {
		return domain.PolicyVersion{}, err
	}
	approvalPrevious := approval.Revision
	if err := approval.Decide(actor.ID, decision, reason, expected, a.now()); err != nil {
		return domain.PolicyVersion{}, err
	}
	policy, err := a.store.GetPolicy(ctx, approval.PolicyVersionID)
	if err != nil {
		return domain.PolicyVersion{}, err
	}
	policyPrevious := policy.Revision
	if decision == domain.DecisionApproved {
		if err := policy.Approve(policy.Revision); err != nil {
			return domain.PolicyVersion{}, err
		}
	} else {
		if policy.Status != domain.PolicyReview {
			return domain.PolicyVersion{}, domain.ErrInvalidTransition
		}
		policy.Status, policy.Revision = domain.PolicyDraft, policy.Revision+1
	}
	audit := a.newAuditEvent(actor, "policy.decide", "policy/"+policy.ID, string(decision), map[string]string{"approval_id": approval.ID})
	if err := a.store.SaveApproval(ctx, approval, approvalPrevious); err != nil {
		return domain.PolicyVersion{}, err
	}
	if err := a.store.SavePolicy(ctx, policy, policyPrevious); err != nil {
		return domain.PolicyVersion{}, err
	}
	if err := a.store.AppendAudit(ctx, audit); err != nil {
		return domain.PolicyVersion{}, err
	}
	return policy, nil
}

func (a *App) ListPolicies(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.PolicyVersion], error) {
	if err := actor.Require("data_admin", "policy_editor", "policy_reviewer", "auditor"); err != nil {
		return domain.Page[domain.PolicyVersion]{}, err
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "name": {}, "version": {}}, map[string]struct{}{"status": {}, "group_id": {}, "scope": {}})
	if err != nil {
		return domain.Page[domain.PolicyVersion]{}, err
	}
	return a.store.ListPolicies(ctx, normalized)
}

func (a *App) ListApprovals(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.Approval], error) {
	if err := actor.Require("data_admin", "policy_reviewer", "auditor"); err != nil {
		return domain.Page[domain.Approval]{}, err
	}
	normalized, err := request.Normalize(
		map[string]struct{}{"created_at": {}, "decision": {}},
		map[string]struct{}{"decision": {}, "policy_version_id": {}, "requested_by": {}, "reviewer": {}},
	)
	if err != nil {
		return domain.Page[domain.Approval]{}, err
	}
	return a.store.ListApprovals(ctx, normalized)
}
