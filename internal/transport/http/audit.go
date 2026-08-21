package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/middleware"
)

func (s *Server) registerAudit(group *gin.RouterGroup) {
	group.GET("/audit", func(c *gin.Context) {
		result, err := s.app.ListAudit(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}
