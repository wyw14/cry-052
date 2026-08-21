package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestPolicySubmissionConflictLeavesPolicyDraft(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	policy := domain.PolicyVersion{ID: "policy-v1", GroupID: "policy-group", Name: "customer protection", Version: 1, Revision: 1, Status: domain.PolicyDraft, Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}, CreatedBy: "editor", CreatedAt: time.Unix(1, 0)}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	conflict := domain.Approval{ID: "approval-conflict", PolicyVersionID: "other-policy", RequestedBy: "other", Decision: domain.DecisionPending, Revision: 1, CreatedAt: time.Unix(1, 0)}
	if err := store.CreateApproval(ctx, conflict); err != nil {
		t.Fatal(err)
	}
	app := New(Dependencies{Store: store, NewID: func() string { return "approval-conflict" }, Now: func() time.Time { return time.Unix(2, 0) }})
	_, err := app.SubmitPolicy(ctx, Actor{ID: "editor", Role: "policy_editor", RequestID: "req-submit"}, policy.ID, policy.Revision)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected approval conflict, got %v", err)
	}
	actual, err := store.GetPolicy(ctx, policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Status != domain.PolicyDraft || actual.Revision != policy.Revision {
		t.Fatalf("submission conflict partially changed policy: status=%s revision=%d", actual.Status, actual.Revision)
	}
}
