package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type requestLifecycle struct {
	duration time.Duration
	started  time.Time
}

func newRequestLifecycle(duration time.Duration) requestLifecycle {
	if duration <= 0 {
		duration = time.Second
	}
	return requestLifecycle{duration: duration, started: time.Now()}
}

func (l requestLifecycle) begin() (context.Context, context.CancelFunc) {
	remaining := l.duration - time.Since(l.started)
	if remaining <= 0 {
		remaining = time.Millisecond
	}
	return context.WithTimeout(context.Background(), remaining)
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		lifecycle := newRequestLifecycle(duration)
		ctx, cancel := lifecycle.begin()
		defer cancel()
		c.Request = c.Request.Clone(ctx)
		c.Next()
	}
}
