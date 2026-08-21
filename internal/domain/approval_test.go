package domain

import (
	"errors"
	"testing"
	"time"
)

func TestApprovalSeparatesRequesterAndReviewer(t *testing.T) {
	approval := Approval{RequestedBy: "author", Decision: DecisionPending, Revision: 1}
	if err := approval.Decide("author", DecisionApproved, "", 1, time.Now()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("requester approved own policy: %v", err)
	}
	if err := approval.Decide("reviewer", DecisionRejected, "", 1, time.Now()); err == nil {
		t.Fatal("rejection without reason should fail")
	}
	if err := approval.Decide("reviewer", DecisionApproved, "", 1, time.Now()); err != nil {
		t.Fatal(err)
	}
}
