package domain

import (
	"path/filepath"
	"strings"
	"time"
)

type LocalRecord struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Attachment struct {
	LocalRecord
	Name          string `json:"name"`
	ContentType   string `json:"content_type"`
	LocalPath     string `json:"-"`
	CreatedBy     string `json:"created_by"`
	StoragePolicy string `json:"storage_policy"`
}

func NewAttachment(id, name, contentType, localPath, actor string, createdAt time.Time) (Attachment, error) {
	cleanName := filepath.Base(strings.TrimSpace(name))
	if id == "" || cleanName == "." || cleanName == "" || contentType == "" || localPath == "" || actor == "" {
		return Attachment{}, NewValidationError("attachment identity is incomplete")
	}
	return Attachment{LocalRecord: LocalRecord{ID: id, CreatedAt: createdAt}, Name: cleanName, ContentType: contentType, LocalPath: localPath, CreatedBy: actor, StoragePolicy: "local-controlled"}, nil
}

type LocalCallback struct {
	LocalRecord
	Topic    string            `json:"topic"`
	Payload  map[string]string `json:"payload"`
	Delivery string            `json:"delivery"`
}

func NewLocalCallback(id, topic string, payload map[string]string, createdAt time.Time) (LocalCallback, error) {
	if id == "" || !strings.Contains(topic, ".") {
		return LocalCallback{}, NewValidationError("callback requires an id and namespaced topic")
	}
	return LocalCallback{LocalRecord: LocalRecord{ID: id, CreatedAt: createdAt}, Topic: topic, Payload: cloneLocalPayload(payload), Delivery: "in_process"}, nil
}

type ScheduledEvent struct {
	LocalRecord
	Name      string            `json:"name"`
	RunAt     time.Time         `json:"run_at"`
	Payload   map[string]string `json:"payload"`
	CreatedBy string            `json:"created_by"`
}

func NewScheduledEvent(id, name string, runAt time.Time, payload map[string]string, actor string, now time.Time) (ScheduledEvent, error) {
	event := ScheduledEvent{LocalRecord: LocalRecord{ID: id, CreatedAt: now}, Name: strings.TrimSpace(name), RunAt: runAt, Payload: cloneLocalPayload(payload), CreatedBy: actor}
	if err := event.Validate(now); err != nil {
		return ScheduledEvent{}, err
	}
	return event, nil
}

func (e ScheduledEvent) Validate(now time.Time) error {
	if e.ID == "" || e.Name == "" || e.CreatedBy == "" {
		return NewValidationError("scheduled event identity is incomplete")
	}
	if !e.RunAt.After(now) {
		return NewValidationError("run_at must be in the future")
	}
	return nil
}

func cloneLocalPayload(payload map[string]string) map[string]string {
	cloned := make(map[string]string, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}
