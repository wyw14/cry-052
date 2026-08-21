package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func TestBatchOptimisticUpdateAllowsSingleConcurrentWriter(t *testing.T) {
	store := New()
	batch := domain.Batch{ID: "b1", IdempotencyKey: "key", Status: domain.BatchPending, Version: 1, CreatedAt: time.Now()}
	if err := store.CreateBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(2)
	var done sync.WaitGroup
	done.Add(2)
	var successes atomic.Int32
	var conflicts atomic.Int32
	for i := 0; i < 2; i++ {
		go func() {
			defer done.Done()
			candidate, err := store.GetBatch(context.Background(), "b1")
			if err != nil {
				t.Error(err)
				return
			}
			ready.Done()
			<-start
			candidate.Status = domain.BatchRunning
			candidate.Version++
			err = store.SaveBatch(context.Background(), candidate, 1)
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, domain.ErrVersionConflict) {
				conflicts.Add(1)
			} else {
				t.Error(err)
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	if successes.Load() != 1 || conflicts.Load() != 1 {
		t.Fatalf("success=%d conflict=%d", successes.Load(), conflicts.Load())
	}
}
