package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/middleware"
	"go.uber.org/zap"
)

type Readiness interface{ Ping(context.Context) error }
type Server struct {
	app       *application.App
	validate  *validator.Validate
	readiness Readiness
}

type UIBootstrap struct {
	Framework string   `json:"framework"`
	Locale    string   `json:"locale"`
	Routes    []string `json:"routes"`
	Version   int      `json:"version"`
}

func newUIBootstrap() UIBootstrap {
	return UIBootstrap{Framework: "vue", Locale: "zh-CN", Routes: []string{"/", "/batches"}, Version: 1}
}

func (s *Server) respond(c *gin.Context, status int, value any, err error) {
	if err != nil {
		writeError(c, err)
		return
	}
	if value == nil {
		c.Status(status)
		return
	}
	c.JSON(status, value)
}

func New(app *application.App, readiness Readiness, auth middleware.Authenticator, logger *zap.Logger, timeout time.Duration) *gin.Engine {
	server := &Server{app: app, validate: validator.New(), readiness: readiness}
	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.SecurityHeaders(), middleware.CORS(map[string]struct{}{"http://localhost:5173": {}}), middleware.Timeout(timeout), middleware.Recovery(logger))
	engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	engine.GET("/readyz", func(c *gin.Context) {
		if err := readiness.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "request_id": middleware.CurrentRequestID(c)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	v1 := engine.Group("/api/v1")
	v1.Use(middleware.Authenticate(auth))
	v1.GET("/ui/bootstrap", func(c *gin.Context) { c.JSON(http.StatusOK, newUIBootstrap()) })
	server.registerDataSources(v1)
	server.registerPolicies(v1)
	server.registerPreviews(v1)
	server.registerBatches(v1)
	server.registerAudit(v1)
	server.registerAttachments(v1)
	server.registerLocalEvents(v1)
	return engine
}
