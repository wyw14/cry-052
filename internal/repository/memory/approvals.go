package memory

import (
	"context"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreateApproval(ctx context.Context, approval domain.Approval) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.approvals[approval.ID]; exists {
		return domain.ErrConflict
	}
	for _, item := range s.approvals {
		if item.PolicyVersionID == approval.PolicyVersionID && item.Decision == domain.DecisionPending {
			return domain.ErrConflict
		}
	}
	s.approvals[approval.ID] = clone(approval)
	return nil
}

func (s *Store) GetApproval(ctx context.Context, id string) (domain.Approval, error) {
	if err := ctx.Err(); err != nil {
		return domain.Approval{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.approvals[id]
	if !ok {
		return domain.Approval{}, domain.ErrNotFound
	}
	return clone(item), nil
}

func (s *Store) SaveApproval(ctx context.Context, approval domain.Approval, expected int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.approvals[approval.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Revision != expected {
		return domain.ErrVersionConflict
	}
	s.approvals[approval.ID] = clone(approval)
	return nil
}
