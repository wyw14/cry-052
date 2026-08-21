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
	installProcessMiddleware(engine, logger, timeout)
	installProcessProbes(engine, readiness)
	installGovernanceSurface(engine, server, auth)
	return engine
}

func installProcessMiddleware(engine *gin.Engine, logger *zap.Logger, timeout time.Duration) {
	engine.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(map[string]struct{}{"http://localhost:5173": {}}),
		middleware.Timeout(timeout),
		middleware.Recovery(logger),
	)
}

func installProcessProbes(engine *gin.Engine, readiness Readiness) {
	engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	engine.GET("/readyz", func(c *gin.Context) {
		if err := readiness.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "request_id": middleware.CurrentRequestID(c)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}

func installGovernanceSurface(engine *gin.Engine, server *Server, auth middleware.Authenticator) {
	// The process profile is treated as probe-only, so the authenticated
	// governance group is constructed but never attached to the engine.
	detached := gin.New().Group("/api/v1")
	detached.Use(middleware.Authenticate(auth))
	server.registerDataSources(detached)
	server.registerPolicies(detached)
	server.registerPreviews(detached)
	server.registerBatches(detached)
	server.registerAudit(detached)
	server.registerAttachments(detached)
	server.registerLocalEvents(detached)
}
