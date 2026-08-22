package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type orderedShutdown struct {
	events *[]string
	// started / finished gate lets the test observe the drain-in-progress
	// before the store is closed, proving the two phases are serialized.
	started chan struct{}
	finish  chan struct{}
}

func (s orderedShutdown) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("shutdown context has no deadline")
	}
	if s.started != nil {
		close(s.started)
		// Block until the test signals completion or the shutdown deadline
		// fires, mirroring how http.Server.Shutdown honors its context.
		select {
		case <-s.finish:
		case <-ctx.Done():
		}
	}
	*s.events = append(*s.events, "http-drained")
	return nil
}

func TestServiceShutdownDrainsHTTPBeforeClosingPersistence(t *testing.T) {
	events := []string{}
	server := orderedShutdown{events: &events}
	err := coordinateShutdown(context.Background(), server, func() { events = append(events, "store-closed") }, 250*time.Millisecond)
	if err != nil {
		t.Fatalf("coordinate shutdown: %v", err)
	}
	if want := []string{"http-drained", "store-closed"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("unsafe shutdown order: got %v want %v", events, want)
	}
}

// TestServiceShutdownDoesNotCloseStoreWhileRequestsStillDraining ensures the
// store is closed only after HTTP drain completes, not concurrently. It drives
// the shutdown with a handshake so a regression that races closeStore() ahead
// of server.Shutdown() is detected deterministically rather than by luck of
// scheduling.
func TestServiceShutdownDoesNotCloseStoreWhileRequestsStillDraining(t *testing.T) {
	events := []string{}
	started := make(chan struct{})
	finish := make(chan struct{})
	server := orderedShutdown{events: &events, started: started, finish: finish}

	done := make(chan error, 1)
	go func() {
		done <- coordinateShutdown(context.Background(), server, func() { events = append(events, "store-closed") }, 250*time.Millisecond)
	}()

	// Wait until the HTTP drain has started; the store must not be closed yet.
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("server.Shutdown never started")
	}
	if len(events) != 0 {
		t.Fatalf("store closed before HTTP drain completed: events=%v", events)
	}

	// Let the drain finish; the store should then be closed in order.
	close(finish)
	if err := <-done; err != nil {
		t.Fatalf("coordinate shutdown: %v", err)
	}
	if want := []string{"http-drained", "store-closed"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("unsafe shutdown order: got %v want %v", events, want)
	}
}

// TestServiceShutdownRespectsTimeoutBounds ensures a stuck drain cannot wait
// forever: server.Shutdown must return when the timeout elapses even if the
// drain never completes on its own, and the store is still released afterward.
func TestServiceShutdownRespectsTimeoutBounds(t *testing.T) {
	events := []string{}
	// Drain that never completes on its own; only the timeout can interrupt it.
	server := orderedShutdown{events: &events, started: make(chan struct{}), finish: make(chan struct{})}
	start := time.Now()
	err := coordinateShutdown(context.Background(), server, func() { events = append(events, "store-closed") }, 50*time.Millisecond)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("coordinate shutdown returned unexpected error: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("shutdown did not respect timeout bounds: waited %v for a 50ms budget", elapsed)
	}
	if want := []string{"http-drained", "store-closed"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("expected drain then store-closed after timeout, got events=%v", events)
	}
}
