package domain

import (
	"fmt"
	"maps"
	"strings"
	"time"
)

type StrategyKind string

const (
	StrategyMask       StrategyKind = "mask"
	StrategyReplace    StrategyKind = "replace"
	StrategyGeneralize StrategyKind = "generalize"
	StrategyHash       StrategyKind = "hash"
	StrategyKeep       StrategyKind = "keep"
)

type PolicyStatus string

const (
	PolicyDraft    PolicyStatus = "draft"
	PolicyReview   PolicyStatus = "review"
	PolicyApproved PolicyStatus = "approved"
	PolicyRetired  PolicyStatus = "retired"
)

type Strategy struct {
	Kind       StrategyKind      `json:"kind"`
	Parameters map[string]string `json:"parameters"`
}

type PolicyVersion struct {
	ID            string       `json:"id"`
	GroupID       string       `json:"group_id"`
	Name          string       `json:"name"`
	Version       int          `json:"version"`
	Status        PolicyStatus `json:"status"`
	Scopes        []string     `json:"scopes"`
	Strategies    []Strategy   `json:"strategies"`
	ChangeSummary string       `json:"change_summary"`
	CreatedBy     string       `json:"created_by"`
	CreatedAt     time.Time    `json:"created_at"`
	Revision      int64        `json:"revision"`
}

func (p PolicyVersion) Validate() error {
	if p.ID == "" || p.GroupID == "" || strings.TrimSpace(p.Name) == "" || p.Version < 1 {
		return NewValidationError("policy version is incomplete")
	}
	if len(p.Strategies) == 0 {
		return NewValidationError("policy has no strategy", FieldViolation{Field: "strategies", Message: "at least one strategy is required"})
	}
	for _, strategy := range p.Strategies {
		if !strategy.Kind.Valid() {
			return fmt.Errorf("unsupported strategy %q", strategy.Kind)
		}
	}
	return nil
}

func (k StrategyKind) Valid() bool {
	switch k {
	case StrategyMask, StrategyReplace, StrategyGeneralize, StrategyHash, StrategyKeep:
		return true
	default:
		return false
	}
}

func (p PolicyVersion) AppliesTo(qualifiedTable string) bool {
	for _, scope := range p.Scopes {
		if strings.EqualFold(strings.TrimSpace(scope), qualifiedTable) {
			return true
		}
	}
	return false
}

func (p PolicyVersion) AllowsStrategy(candidate Strategy) bool {
	for _, strategy := range p.Strategies {
		if strategy.Kind == candidate.Kind && maps.Equal(strategy.Parameters, candidate.Parameters) {
			return true
		}
	}
	return false
}

func (p *PolicyVersion) Submit(expected int64) error {
	if p.Revision != expected {
		return ErrVersionConflict
	}
	if p.Status != PolicyDraft {
		return ErrInvalidTransition
	}
	p.Status = PolicyReview
	p.Revision++
	return nil
}

func (p *PolicyVersion) Approve(expected int64) error {
	if p.Revision != expected {
		return ErrVersionConflict
	}
	if p.Status != PolicyReview {
		return ErrInvalidTransition
	}
	p.Status = PolicyApproved
	p.Revision++
	return nil
}
