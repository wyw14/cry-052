package memory

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

func (s *Store) CreatePreview(ctx context.Context, preview domain.Preview) error {
	if err := validatePreviewWrite(ctx, preview); err != nil {
		return err
	}
	snapshot := clone(preview)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, occupied := s.previews[snapshot.ID]; occupied {
		return fmt.Errorf("preview %s: %w", snapshot.ID, domain.ErrConflict)
	}
	s.previews[snapshot.ID] = snapshot
	return nil
}

func (s *Store) GetPreview(ctx context.Context, id string) (domain.Preview, error) {
	if err := ctx.Err(); err != nil {
		return domain.Preview{}, err
	}
	s.mu.RLock()
	preview, found := s.previews[id]
	s.mu.RUnlock()
	if !found {
		return domain.Preview{}, fmt.Errorf("preview %s: %w", id, domain.ErrNotFound)
	}
	return clone(preview), nil
}

func (s *Store) SavePreview(ctx context.Context, preview domain.Preview) error {
	if err := validatePreviewWrite(ctx, preview); err != nil {
		return err
	}
	snapshot := clone(preview)
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, found := s.previews[snapshot.ID]
	if !found {
		return fmt.Errorf("preview %s: %w", snapshot.ID, domain.ErrNotFound)
	}
	if previous.ConfirmedAt != nil && (snapshot.ConfirmedAt == nil || previous.ConfirmedBy != snapshot.ConfirmedBy) {
		return fmt.Errorf("preview confirmation is immutable: %w", domain.ErrConflict)
	}
	s.previews[snapshot.ID] = snapshot
	return nil
}

func validatePreviewWrite(ctx context.Context, preview domain.Preview) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if preview.ID == "" || preview.SourceTableID == "" || preview.TargetTableID == "" {
		return domain.NewValidationError("preview identity and table references are required")
	}
	if preview.SourceTableID == preview.TargetTableID {
		return domain.ErrSourceOverwrite
	}
	return nil
}
