package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
)

func TestPolicyScopeAndScheduleDefaultListContracts(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Now().UTC()
	policy := domain.PolicyVersion{ID: "policy", GroupID: "group", Name: "Scoped", Version: 1, Status: domain.PolicyApproved, Scopes: []string{"sample.customers", "sample.contacts"}, Strategies: []domain.Strategy{{Kind: domain.StrategyMask}}, Revision: 3, CreatedAt: now}
	if err := store.CreatePolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	scheduler := platform.NewLocalScheduler()
	app := New(Dependencies{Store: store, Scheduler: scheduler, Redactor: platform.NewRedactor()})
	admin := Actor{ID: "admin", Role: "data_admin", RequestID: "request-list"}
	matching, err := app.ListPolicies(ctx, admin, domain.PageRequest{Filters: map[string]string{"scope": "sample.contacts"}})
	if err != nil || matching.Total != 1 {
		t.Fatalf("matching=%+v err=%v", matching, err)
	}
	notMatching, err := app.ListPolicies(ctx, admin, domain.PageRequest{Filters: map[string]string{"scope": "sample.other"}})
	if err != nil || notMatching.Total != 0 {
		t.Fatalf("notMatching=%+v err=%v", notMatching, err)
	}
	if _, err := app.ScheduleLocalEvent(ctx, admin, "nightly-preview", now.Add(time.Hour), map[string]string{"policy_id": policy.ID}); err != nil {
		t.Fatal(err)
	}
	page, err := app.ListScheduledEvents(ctx, admin, domain.PageRequest{})
	if err != nil || page.Total != 1 || page.Items[0].Name != "nightly-preview" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}
