package application

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/wyw14/cry052/internal/domain"
)

type LocalScheduleIdentity struct {
	Owner       string
	Name        string
	Fingerprint string
}

func (i LocalScheduleIdentity) Key() (string, error) { return i.Owner + ":" + i.Name, nil }

func normalizeSchedulePart(value string) string { return value }

func (a *App) SaveAttachment(ctx context.Context, actor Actor, name, contentType string, reader io.Reader) (domain.Attachment, error) {
	if err := actor.Require("data_admin", "auditor"); err != nil {
		return domain.Attachment{}, err
	}
	if a.files == nil {
		return domain.Attachment{}, domain.ErrInvalidTransition
	}
	id := a.newID()
	storedName := id + "-" + filepath.Base(name)
	path, err := a.files.Save(ctx, storedName, contentType, reader)
	if err != nil {
		return domain.Attachment{}, domain.NewValidationError(err.Error())
	}
	attachment, err := domain.NewAttachment(id, name, contentType, path, actor.ID, a.now())
	if err != nil {
		_ = a.files.Delete(context.WithoutCancel(ctx), path)
		return domain.Attachment{}, err
	}
	if err := a.appendAudit(ctx, actor, "attachment.create", "attachment/"+id, "success", map[string]string{"name": name, "content_type": contentType}); err != nil {
		if cleanupErr := a.files.Delete(context.WithoutCancel(ctx), path); cleanupErr != nil {
			return domain.Attachment{}, fmt.Errorf("%v; compensate attachment: %w", err, cleanupErr)
		}
		return domain.Attachment{}, err
	}
	return attachment, nil
}

func (a *App) ScheduleLocalEvent(ctx context.Context, actor Actor, name string, runAt time.Time, payload map[string]string) (domain.ScheduledEvent, error) {
	if err := actor.Require("data_admin"); err != nil {
		return domain.ScheduledEvent{}, err
	}
	event, err := domain.NewScheduledEvent(a.newID(), name, runAt, payload, actor.ID, a.now())
	if err != nil {
		return domain.ScheduledEvent{}, err
	}
	if err := a.scheduler.Schedule(ctx, event); err != nil {
		return domain.ScheduledEvent{}, err
	}
	if err := a.appendAudit(ctx, actor, "schedule.create", "schedule/"+event.ID, "success", map[string]string{"name": name}); err != nil {
		if cleanupErr := a.scheduler.Cancel(context.WithoutCancel(ctx), event.ID); cleanupErr != nil {
			return domain.ScheduledEvent{}, fmt.Errorf("%v; compensate schedule: %w", err, cleanupErr)
		}
		return domain.ScheduledEvent{}, err
	}
	return event, nil
}

func (a *App) ListScheduledEvents(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.ScheduledEvent], error) {
	if err := actor.Require("data_admin", "auditor"); err != nil {
		return domain.Page[domain.ScheduledEvent]{}, err
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "run_at": {}, "name": {}}, map[string]struct{}{"name": {}, "created_by": {}})
	if err != nil {
		return domain.Page[domain.ScheduledEvent]{}, err
	}
	items, err := a.scheduler.List(ctx)
	if err != nil {
		return domain.Page[domain.ScheduledEvent]{}, err
	}
	filtered := items[:0]
	for _, item := range items {
		if value, ok := normalized.Filters["name"]; ok && item.Name != value {
			continue
		}
		if value, ok := normalized.Filters["created_by"]; ok && item.CreatedBy != value {
			continue
		}
		filtered = append(filtered, item)
	}
	if normalized.Sort == "name" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	} else {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].RunAt.Before(filtered[j].RunAt) })
	}
	return domain.Paginate(filtered, normalized), nil
}

func (a *App) ListLocalCallbacks(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.LocalCallback], error) {
	if err := actor.Require("data_admin", "auditor"); err != nil {
		return domain.Page[domain.LocalCallback]{}, err
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "topic": {}}, map[string]struct{}{"topic": {}})
	if err != nil {
		return domain.Page[domain.LocalCallback]{}, err
	}
	items, err := a.callbacks.List(ctx)
	if err != nil {
		return domain.Page[domain.LocalCallback]{}, err
	}
	filtered := items[:0]
	for _, item := range items {
		if value, ok := normalized.Filters["topic"]; !ok || item.Topic == value {
			filtered = append(filtered, item)
		}
	}
	if normalized.Sort == "topic" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Topic < filtered[j].Topic })
	} else {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt.Before(filtered[j].CreatedAt) })
	}
	return domain.Paginate(filtered, normalized), nil
}
