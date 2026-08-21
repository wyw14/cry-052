package platform

import (
	"container/heap"
	"context"
	"sort"
	"sync"

	"github.com/wyw14/cry052/internal/domain"
)

type queuedEvent struct {
	event domain.ScheduledEvent
	index int
}

type eventQueue []*queuedEvent

func (q eventQueue) Len() int           { return len(q) }
func (q eventQueue) Less(i, j int) bool { return q[i].event.RunAt.Before(q[j].event.RunAt) }
func (q eventQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index, q[j].index = i, j
}
func (q *eventQueue) Push(value any) {
	item := value.(*queuedEvent)
	item.index = len(*q)
	*q = append(*q, item)
}
func (q *eventQueue) Pop() any {
	old := *q
	last := len(old) - 1
	item := old[last]
	old[last] = nil
	item.index = -1
	*q = old[:last]
	return item
}

type LocalScheduler struct {
	mu    sync.RWMutex
	queue eventQueue
	byID  map[string]*queuedEvent
}

func NewLocalScheduler() *LocalScheduler {
	scheduler := &LocalScheduler{byID: make(map[string]*queuedEvent)}
	heap.Init(&scheduler.queue)
	return scheduler
}

func (s *LocalScheduler) Schedule(ctx context.Context, event domain.ScheduledEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	item := &queuedEvent{event: cloneScheduledEvent(event)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byID[event.ID]; exists {
		return domain.ErrConflict
	}
	heap.Push(&s.queue, item)
	s.byID[event.ID] = item
	return nil
}

func (s *LocalScheduler) Cancel(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, exists := s.byID[id]
	if !exists {
		return domain.ErrNotFound
	}
	heap.Remove(&s.queue, item.index)
	delete(s.byID, id)
	return nil
}

func (s *LocalScheduler) List(ctx context.Context) ([]domain.ScheduledEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	result := make([]domain.ScheduledEvent, 0, len(s.byID))
	for _, item := range s.byID {
		result = append(result, cloneScheduledEvent(item.event))
	}
	s.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		if result[i].RunAt.Equal(result[j].RunAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].RunAt.Before(result[j].RunAt)
	})
	return result, nil
}

func cloneScheduledEvent(event domain.ScheduledEvent) domain.ScheduledEvent {
	event.Payload = cloneStringMap(event.Payload)
	return event
}
