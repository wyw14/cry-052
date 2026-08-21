package platform

import (
	"context"
	"sync"
	"time"
)

type Notice struct {
	Topic, Message string
	Metadata       map[string]string
	CreatedAt      time.Time
}
type LocalNotifier struct {
	mu      sync.Mutex
	notices []Notice
	now     func() time.Time
}

func NewLocalNotifier(now func() time.Time) *LocalNotifier {
	if now == nil {
		now = time.Now
	}
	return &LocalNotifier{now: now}
}
func (n *LocalNotifier) Notify(ctx context.Context, topic, message string, metadata map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	copyMeta := map[string]string{}
	for k, v := range metadata {
		copyMeta[k] = v
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notices = append(n.notices, Notice{Topic: topic, Message: message, Metadata: copyMeta, CreatedAt: n.now()})
	return nil
}
func (n *LocalNotifier) Notices() []Notice {
	n.mu.Lock()
	defer n.mu.Unlock()
	result := make([]Notice, len(n.notices))
	for index, notice := range n.notices {
		metadata := make(map[string]string, len(notice.Metadata))
		for key, value := range notice.Metadata {
			metadata[key] = value
		}
		result[index] = notice
		result[index].Metadata = metadata
	}
	return result
}
