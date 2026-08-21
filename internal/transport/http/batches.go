package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
)

type createBatchRequest struct {
	PreviewID string `json:"preview_id" validate:"required"`
}
type runBatchRequest struct {
	Mappings []domain.FieldMapping `json:"mappings" validate:"required,min=1"`
}
type versionedBatchRequest struct {
	Version int64 `json:"version" validate:"gte=1"`
}
type recoverBatchRequest struct {
	Version          int64  `json:"version" validate:"gte=1"`
	InputFingerprint string `json:"input_fingerprint" validate:"required"`
}

func (s *Server) registerBatches(group *gin.RouterGroup) {
	batches := group.Group("/batches")
	batches.POST("", func(c *gin.Context) {
		var request createBatchRequest
		if !s.bindAndValidate(c, &request) {
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			s.respond(c, http.StatusBadRequest, nil, domain.NewValidationError("Idempotency-Key is required"))
			return
		}
		batch, err := s.app.CreateBatch(c.Request.Context(), middleware.CurrentActor(c), application.CreateBatch{PreviewID: request.PreviewID, IdempotencyKey: key})
		s.respond(c, http.StatusCreated, batch, err)
	})
	batches.GET("", func(c *gin.Context) {
		page, err := s.app.ListBatches(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		s.respond(c, http.StatusOK, page, err)
	})
	batches.POST("/:id/run", func(c *gin.Context) {
		var request runBatchRequest
		if s.bindAndValidate(c, &request) {
			err := s.app.RunBatch(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), request.Mappings)
			s.respond(c, http.StatusAccepted, nil, err)
		}
	})
	batches.POST("/:id/cancel", func(c *gin.Context) {
		var request versionedBatchRequest
		if s.bindAndValidate(c, &request) {
			err := s.app.CancelBatch(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), request.Version)
			s.respond(c, http.StatusAccepted, nil, err)
		}
	})
	batches.POST("/:id/recover", func(c *gin.Context) {
		var request recoverBatchRequest
		if s.bindAndValidate(c, &request) {
			batch, err := s.app.RecoverBatch(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), request.InputFingerprint, request.Version)
			s.respond(c, http.StatusOK, batch, err)
		}
	})
	batches.POST("/:id/rollback", func(c *gin.Context) {
		batch, err := s.app.RollbackBatch(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"))
		s.respond(c, http.StatusOK, batch, err)
	})
	batches.GET("/:id/report", func(c *gin.Context) {
		payload, contentType, err := s.app.ExportReport(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), c.DefaultQuery("format", "json"))
		if err == nil {
			c.Data(http.StatusOK, contentType, payload)
		} else {
			s.respond(c, http.StatusOK, nil, err)
		}
	})
}

func (s *Server) bindAndValidate(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		writeError(c, err)
		return false
	}
	if err := s.validate.Struct(target); err != nil {
		writeError(c, err)
		return false
	}
	return true
}
