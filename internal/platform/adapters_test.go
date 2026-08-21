package platform

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

func TestLocalAdaptersValidateAndRemainOffline(t *testing.T) {
	files, err := NewFileStore(t.TempDir(), 8, "text/csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := files.Save(context.Background(), "bad.exe", "application/octet-stream", strings.NewReader("x")); err == nil {
		t.Fatal("unexpected content type accepted")
	}
	if _, err := files.Save(context.Background(), "large.csv", "text/csv", bytes.NewBufferString("123456789")); err == nil {
		t.Fatal("oversized attachment accepted")
	}
	if path, err := files.Save(context.Background(), "ok.csv", "text/csv", strings.NewReader("a,b\n")); err != nil || path == "" {
		t.Fatalf("save path=%q err=%v", path, err)
	}
	redacted := NewRedactor().Fields(map[string]string{"dsn": "postgres://user:pass@localhost/db", "note": "token=abc123"})
	if redacted["dsn"] != "[redacted]" || strings.Contains(redacted["note"], "abc123") {
		t.Fatalf("redaction=%+v", redacted)
	}
	callback := NewLocalCallbackSink()
	event, err := domain.NewLocalCallback("cb", "batch.completed", map[string]string{"batch_id": "b"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := callback.Deliver(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	events, _ := callback.List(context.Background())
	events[0].Payload["batch_id"] = "mutated"
	again, _ := callback.List(context.Background())
	if again[0].Payload["batch_id"] != "b" {
		t.Fatal("callback leaked mutable state")
	}
	scheduler := NewLocalScheduler()
	scheduled, err := domain.NewScheduledEvent("s", "nightly", time.Now().Add(time.Hour), map[string]string{"source": "local"}, "admin", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Schedule(context.Background(), scheduled); err != nil {
		t.Fatal(err)
	}
	list, _ := scheduler.List(context.Background())
	if len(list) != 1 || list[0].Name != "nightly" {
		t.Fatalf("schedules=%+v", list)
	}
}

func TestRedactorBuildsSafeStructuredErrorField(t *testing.T) {
	field := NewRedactor().Error(fmt.Errorf("connect postgres://user:password@localhost/db token=secret-value"))
	if strings.Contains(field.String, "password") || strings.Contains(field.String, "secret-value") {
		t.Fatalf("structured log field leaked secret: %s", field.String)
	}
}

func TestLocalNotifierPublishesImmutableNoticeSnapshots(t *testing.T) {
	now := time.Date(2026, time.August, 21, 9, 30, 0, 0, time.UTC)
	notifier := NewLocalNotifier(func() time.Time { return now })
	metadata := map[string]string{"batch_id": "batch-52", "status": "completed"}
	if err := notifier.Notify(context.Background(), "batch.completed", "脱敏批次已完成", metadata); err != nil {
		t.Fatal(err)
	}
	metadata["status"] = "mutated-input"

	notices := notifier.Notices()
	if len(notices) != 1 || notices[0].Topic != "batch.completed" || notices[0].Message != "脱敏批次已完成" || notices[0].CreatedAt != now {
		t.Fatalf("notice=%+v", notices)
	}
	if notices[0].Metadata["status"] != "completed" || notices[0].Metadata["batch_id"] != "batch-52" {
		t.Fatalf("metadata=%+v", notices[0].Metadata)
	}
	notices[0].Metadata["status"] = "mutated-output"
	if notifier.Notices()[0].Metadata["status"] != "completed" {
		t.Fatal("notifier leaked mutable metadata through Notices")
	}
}
