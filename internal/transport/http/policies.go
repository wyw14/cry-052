package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
)

type createPolicyRequest struct {
	GroupID       string            `json:"group_id"`
	Name          string            `json:"name" validate:"required,min=2"`
	Version       int               `json:"version" validate:"gte=1"`
	Scopes        []string          `json:"scopes" validate:"required,min=1"`
	Strategies    []domain.Strategy `json:"strategies" validate:"required,min=1"`
	ChangeSummary string            `json:"change_summary" validate:"required"`
}

func (s *Server) registerPolicies(group *gin.RouterGroup) {
	group.POST("/policies", func(c *gin.Context) {
		var input createPolicyRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, err)
			return
		}
		if err := s.validate.Struct(input); err != nil {
			writeError(c, err)
			return
		}
		result, err := s.app.CreatePolicy(c.Request.Context(), middleware.CurrentActor(c), domain.PolicyVersion{GroupID: input.GroupID, Name: input.Name, Version: input.Version, Scopes: input.Scopes, Strategies: input.Strategies, ChangeSummary: input.ChangeSummary})
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, result)
	})
	group.GET("/policies", func(c *gin.Context) {
		result, err := s.app.ListPolicies(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/policies/:id/submit", func(c *gin.Context) {
		var input struct {
			Revision int64 `json:"revision" validate:"gte=1"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, err)
			return
		}
		if err := s.validate.Struct(input); err != nil {
			writeError(c, err)
			return
		}
		result, err := s.app.SubmitPolicy(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), input.Revision)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, result)
	})
	group.POST("/approvals/:id/decision", func(c *gin.Context) {
		var input struct {
			Decision domain.ApprovalDecision `json:"decision" validate:"required,oneof=approved rejected"`
			Reason   string                  `json:"reason"`
			Revision int64                   `json:"revision" validate:"gte=1"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, err)
			return
		}
		if err := s.validate.Struct(input); err != nil {
			writeError(c, err)
			return
		}
		result, err := s.app.DecidePolicy(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), input.Decision, input.Reason, input.Revision)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.GET("/approvals", func(c *gin.Context) {
		result, err := s.app.ListApprovals(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}
