package domain

import (
	"testing"
	"time"
)

func TestBatchAdmissionBindsConfirmationActorTimeAndFingerprint(t *testing.T) {
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	confirmedAt := now.Add(-time.Minute)
	preview := Preview{ID: "preview-17", SourceTableID: "source", TargetTableID: "target", InputFingerprint: "fp-approved", ConfirmedBy: "admin-a", ConfirmedAt: &confirmedAt}
	receipt, err := preview.Receipt()
	if err != nil {
		t.Fatalf("confirmation receipt: %v", err)
	}
	if receipt.PreviewID != preview.ID || receipt.ConfirmedBy != "admin-a" || !receipt.ConfirmedAt.Equal(confirmedAt) || receipt.Fingerprint != "fp-approved" {
		t.Errorf("receipt lost confirmation evidence: %#v", receipt)
	}

	preview.InputFingerprint = "fp-mutated-after-confirmation"
	_, err = (BatchAdmission{Preview: preview, Receipt: receipt, Actor: "admin-a", Now: now}).Create("batch-17", "public.source", "masked.target", "key-17")
	if err == nil {
		t.Fatal("batch admission accepted a preview changed after confirmation")
	}

	preview.InputFingerprint = "fp-approved"
	batch, err := (BatchAdmission{Preview: preview, Receipt: receipt, Actor: "admin-a", Now: now}).Create("batch-17", "public.source", "masked.target", "key-17")
	if err != nil {
		t.Fatalf("valid admission rejected: %v", err)
	}
	if batch.InputFingerprint != receipt.Fingerprint || batch.CreatedBy != receipt.ConfirmedBy {
		t.Errorf("batch did not inherit confirmation evidence: %#v", batch)
	}
}
