package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
)

func (s *Server) registerPreviews(group *gin.RouterGroup) {
	group.POST("/previews", func(c *gin.Context) {
		var input struct {
			SourceTableID   string                `json:"source_table_id" validate:"required"`
			TargetTableID   string                `json:"target_table_id" validate:"required"`
			PolicyVersionID string                `json:"policy_version_id" validate:"required"`
			Mappings        []domain.FieldMapping `json:"mappings" validate:"required,min=1"`
			Limit           int                   `json:"limit" validate:"gte=1,lte=100"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, err)
			return
		}
		if err := s.validate.Struct(input); err != nil {
			writeError(c, err)
			return
		}
		result, err := s.app.CreatePreview(c.Request.Context(), middleware.CurrentActor(c), application.CreatePreview{SourceTableID: input.SourceTableID, TargetTableID: input.TargetTableID, PolicyVersionID: input.PolicyVersionID, Mappings: input.Mappings, Limit: input.Limit})
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, result)
	})
	group.POST("/previews/:id/confirm", func(c *gin.Context) {
		result, err := s.app.ConfirmPreview(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}
