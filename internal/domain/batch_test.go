package domain

import (
	"errors"
	"testing"
	"time"
)

func TestBatchRequiresConfirmedPreviewAndSeparateTarget(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	preview := Preview{ID: "p1", InputFingerprint: "fp"}
	if _, err := NewBatch("b1", preview, "public.people", "masked.people", "key", "admin", now); !errors.Is(err, ErrPreviewRequired) {
		t.Fatalf("expected preview requirement, got %v", err)
	}
	preview.ConfirmedBy = "admin"
	preview.ConfirmedAt = &now
	if _, err := NewBatch("b1", preview, "public.people", "public.people", "key", "admin", now); !errors.Is(err, ErrSourceOverwrite) {
		t.Fatalf("expected source overwrite rejection, got %v", err)
	}
}

func TestBatchRecoveryRequiresMatchingInputSnapshot(t *testing.T) {
	now := time.Now()
	batch := Batch{ID: "b", Status: BatchFailed, InputFingerprint: "stable", Version: 4}
	if err := batch.Recover(4, "changed", now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected changed snapshot rejection, got %v", err)
	}
	if err := batch.Recover(4, "stable", now); err != nil {
		t.Fatal(err)
	}
	if batch.Status != BatchRecovering || batch.Version != 5 {
		t.Fatalf("unexpected recovery state: %+v", batch)
	}
}

func TestPendingBatchCancellationReachesTerminalStateImmediately(t *testing.T) {
	batch := Batch{ID: "pending", Status: BatchPending, Version: 1}
	if err := batch.RequestCancel(1, time.Now()); err != nil {
		t.Fatal(err)
	}
	if batch.Status != BatchCancelled || batch.Version != 2 {
		t.Fatalf("batch=%+v", batch)
	}
}
