package platform

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func TestConcurrentScheduleOncePublishesSingleSharedEvent(t *testing.T) {
	scheduler := NewLocalScheduler()
	start := make(chan struct{})
	var ready, joined sync.WaitGroup
	ready.Add(2)
	joined.Add(2)
	type result struct {
		event   domain.ScheduledEvent
		created bool
		err     error
	}
	results := make(chan result, 2)
	for index := 0; index < 2; index++ {
		index := index
		go func() {
			defer joined.Done()
			ready.Done()
			<-start
			event := domain.ScheduledEvent{LocalRecord: domain.LocalRecord{ID: []string{"event-a", "event-b"}[index]}, Name: "nightly masking", RunAt: time.Date(2026, 8, 22, 1, 0, 0, 0, time.UTC), Payload: map[string]string{"policy": "approved"}}
			got, created, err := scheduler.ScheduleOnce(context.Background(), "admin:nightly-masking", event)
			results <- result{got, created, err}
		}()
	}
	ready.Wait()
	close(start)
	joined.Wait()
	close(results)
	createdCount := 0
	returnedID := ""
	for item := range results {
		if item.err != nil {
			t.Fatalf("schedule once: %v", item.err)
		}
		if item.created {
			createdCount++
		}
		if returnedID == "" {
			returnedID = item.event.ID
		} else if returnedID != item.event.ID {
			t.Errorf("contenders observed different shared events: %q and %q", returnedID, item.event.ID)
		}
	}
	events, err := scheduler.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if createdCount != 1 || len(events) != 1 {
		t.Fatalf("shared schedule duplicated: created=%d events=%#v", createdCount, events)
	}
}
