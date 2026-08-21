package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type orderedShutdown struct{ events *[]string }

func (s orderedShutdown) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("shutdown context has no deadline")
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
