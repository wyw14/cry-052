package domain

import "time"

type ApprovalDecision string

const (
	DecisionPending  ApprovalDecision = "pending"
	DecisionApproved ApprovalDecision = "approved"
	DecisionRejected ApprovalDecision = "rejected"
)

type Approval struct {
	ID              string           `json:"id"`
	PolicyVersionID string           `json:"policy_version_id"`
	RequestedBy     string           `json:"requested_by"`
	Reviewer        string           `json:"reviewer,omitempty"`
	Decision        ApprovalDecision `json:"decision"`
	Reason          string           `json:"reason,omitempty"`
	Revision        int64            `json:"revision"`
	CreatedAt       time.Time        `json:"created_at"`
	DecidedAt       *time.Time       `json:"decided_at,omitempty"`
}

func (a *Approval) Decide(actor string, decision ApprovalDecision, reason string, expected int64, now time.Time) error {
	if a.Revision != expected {
		return ErrVersionConflict
	}
	if a.Decision != DecisionPending || actor == "" || actor == a.RequestedBy {
		return ErrInvalidTransition
	}
	if decision != DecisionApproved && decision != DecisionRejected {
		return NewValidationError("decision must be approved or rejected")
	}
	if decision == DecisionRejected && reason == "" {
		return NewValidationError("rejection reason is required")
	}
	a.Reviewer = actor
	a.Decision = decision
	a.Reason = reason
	a.Revision++
	a.DecidedAt = &now
	return nil
}
