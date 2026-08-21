package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

func TestSubmitPolicyTransactionLeavesPolicyUnchangedOnApprovalConflict(t *testing.T) {
	ctx := context.Background()
	store := New()
	policy := domain.PolicyVersion{ID: "policy", GroupID: "group", Name: "保护", Version: 1, Status: domain.PolicyDraft, Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}, Revision: 1, CreatedAt: time.Now()}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	existing := domain.Approval{ID: "existing", PolicyVersionID: policy.ID, Decision: domain.DecisionPending, Revision: 1, CreatedAt: time.Now()}
	if err := store.CreateApproval(ctx, existing); err != nil {
		t.Fatal(err)
	}
	updated := policy
	if err := updated.Submit(1); err != nil {
		t.Fatal(err)
	}
	audit := domain.AuditEvent{ID: "audit", RequestID: "request", Actor: "admin", Action: "policy.submit", Resource: "policy/" + policy.ID, Outcome: "success", CreatedAt: time.Now()}
	conflicting := domain.Approval{ID: "new", PolicyVersionID: policy.ID, Decision: domain.DecisionPending, Revision: 1}
	err := store.ApplyMutation(ctx, persistence.Mutation{UpdatePolicy: &persistence.Versioned[domain.PolicyVersion]{Value: updated, Expected: 1}, CreateApproval: &conflicting, Audit: audit})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	actual, _ := store.GetPolicy(ctx, policy.ID)
	if actual.Status != domain.PolicyDraft || actual.Revision != 1 {
		t.Fatalf("partial commit: %+v", actual)
	}
}
