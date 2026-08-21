package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

func TestPostgresPolicySubmissionRollsBackWhenApprovalConflicts(t *testing.T) {
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().Format("150405.000000")
	policy := domain.PolicyVersion{ID: "policy-" + suffix, GroupID: "group-" + suffix, Name: "事务策略", Version: 1, Status: domain.PolicyDraft, Strategies: []domain.Strategy{{Kind: domain.StrategyKeep}}, Revision: 1, CreatedAt: time.Now()}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	existing := domain.Approval{ID: "approval-" + suffix, PolicyVersionID: policy.ID, Decision: domain.DecisionPending, Revision: 1, CreatedAt: time.Now()}
	if err := store.CreateApproval(ctx, existing); err != nil {
		t.Fatal(err)
	}
	updated := policy
	if err := updated.Submit(1); err != nil {
		t.Fatal(err)
	}
	audit := domain.AuditEvent{ID: "submit-audit-" + suffix, RequestID: "request-" + suffix, Actor: "admin", Action: "policy.submit", Resource: "policy/" + policy.ID, Outcome: "success", CreatedAt: time.Now()}
	conflicting := domain.Approval{ID: "other-" + suffix, PolicyVersionID: policy.ID, Decision: domain.DecisionPending, Revision: 1}
	err = store.ApplyMutation(ctx, persistence.Mutation{UpdatePolicy: &persistence.Versioned[domain.PolicyVersion]{Value: updated, Expected: 1}, CreateApproval: &conflicting, Audit: audit})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	actual, err := store.GetPolicy(ctx, policy.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Status != domain.PolicyDraft || actual.Revision != 1 {
		t.Fatalf("transaction partially committed: %+v", actual)
	}
}

func TestPostgresMutationRollsBackBusinessWriteWhenAuditConflicts(t *testing.T) {
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().Format("150405.000000000")
	audit := domain.AuditEvent{ID: "duplicate-audit-" + suffix, RequestID: "request-" + suffix, Actor: "admin", Action: "datasource.create", Resource: "datasource/rollback", Outcome: "success", CreatedAt: time.Now()}
	if err := store.AppendAudit(ctx, audit); err != nil {
		t.Fatal(err)
	}
	datasource, err := domain.NewDataSource("rollback-ds-"+suffix, "Rollback "+suffix, domain.DataSourceSample, "secret/rollback", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	err = store.ApplyMutation(ctx, persistence.Mutation{CreateDataSource: &datasource, Audit: audit})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected audit conflict, got %v", err)
	}
	if _, err := store.GetDataSource(ctx, datasource.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("business write was not rolled back: %v", err)
	}
}
