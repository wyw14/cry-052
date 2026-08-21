package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/middleware"
)

func (s *Server) registerAttachments(group *gin.RouterGroup) {
	group.POST("/attachments", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			writeError(c, err)
			return
		}
		opened, err := file.Open()
		if err != nil {
			writeError(c, err)
			return
		}
		defer opened.Close()
		result, err := s.app.SaveAttachment(c.Request.Context(), middleware.CurrentActor(c), file.Filename, file.Header.Get("Content-Type"), opened)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, result)
	})
}

func (s *Server) registerLocalEvents(group *gin.RouterGroup) {
	group.GET("/local-events/callbacks", func(c *gin.Context) {
		result, err := s.app.ListLocalCallbacks(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.GET("/local-events/schedules", func(c *gin.Context) {
		result, err := s.app.ListScheduledEvents(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
	group.POST("/local-events/schedules", func(c *gin.Context) {
		var input struct {
			Name    string            `json:"name" validate:"required"`
			RunAt   time.Time         `json:"run_at" validate:"required"`
			Payload map[string]string `json:"payload"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			writeError(c, err)
			return
		}
		if err := s.validate.Struct(input); err != nil {
			writeError(c, err)
			return
		}
		result, err := s.app.ScheduleLocalEvent(c.Request.Context(), middleware.CurrentActor(c), input.Name, input.RunAt, input.Payload)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, result)
	})
}
