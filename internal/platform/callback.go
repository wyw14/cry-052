package platform

import (
	"context"
	"sync"

	"github.com/wyw14/cry052/internal/domain"
)

type LocalCallbackSink struct {
	mu     sync.RWMutex
	events []domain.LocalCallback
}

func NewLocalCallbackSink() *LocalCallbackSink { return &LocalCallbackSink{} }

func (s *LocalCallbackSink) Deliver(ctx context.Context, event domain.LocalCallback) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.events = append(s.events, cloneCallback(event))
	s.mu.Unlock()
	return nil
}

func (s *LocalCallbackSink) List(ctx context.Context) ([]domain.LocalCallback, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.LocalCallback, len(s.events))
	for i, event := range s.events {
		result[i] = cloneCallback(event)
	}
	return result, nil
}

func cloneCallback(event domain.LocalCallback) domain.LocalCallback {
	event.Payload = cloneStringMap(event.Payload)
	return event
}

func cloneStringMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
