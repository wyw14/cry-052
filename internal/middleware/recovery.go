package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer recoverRequest(c, logger)
		c.Next()
	}
}

func recoverRequest(c *gin.Context, logger *zap.Logger) {
	recovered := recover()
	if recovered == nil {
		return
	}
	requestID := CurrentRequestID(c)
	logger.Error(
		"request panic recovered",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("route", c.FullPath()),
		zap.ByteString("stack", debug.Stack()),
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"code":       "INTERNAL_ERROR",
		"message":    "internal server error",
		"request_id": requestID,
	})
}
