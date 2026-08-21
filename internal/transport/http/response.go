package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
)

type errorResponse struct {
	Code      string                  `json:"code"`
	Message   string                  `json:"message"`
	Fields    []domain.FieldViolation `json:"fields,omitempty"`
	RequestID string                  `json:"request_id"`
}

func writeError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	var fields []domain.FieldViolation
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "resource not found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrVersionConflict), errors.Is(err, domain.ErrIdempotencyReuse), errors.Is(err, domain.ErrInputSnapshotChanged):
		status, code, message = http.StatusConflict, "CONFLICT", err.Error()
	case errors.Is(err, domain.ErrPermissionDenied):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "permission denied"
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrPreviewRequired), errors.Is(err, domain.ErrSourceOverwrite):
		status, code, message = http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION", err.Error()
	case errors.Is(err, io.EOF), errors.Is(err, http.ErrMissingFile):
		status, code, message = http.StatusBadRequest, "VALIDATION_FAILED", "request body is invalid"
	}
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		status, code, message = http.StatusBadRequest, "VALIDATION_FAILED", "request JSON is invalid"
	}
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		status, code, message, fields = http.StatusBadRequest, validationError.Code, validationError.Message, validationError.Violations
	}
	var binding validator.ValidationErrors
	if errors.As(err, &binding) {
		status, code, message = http.StatusBadRequest, "VALIDATION_FAILED", "request validation failed"
		for _, item := range binding {
			fields = append(fields, domain.FieldViolation{Field: item.Field(), Message: item.Tag()})
		}
	}
	c.AbortWithStatusJSON(status, errorResponse{Code: code, Message: message, Fields: fields, RequestID: middleware.CurrentRequestID(c)})
}

func pageRequest(c *gin.Context) domain.PageRequest {
	page, size := 1, 20
	_, _ = fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &page)
	_, _ = fmt.Sscanf(c.DefaultQuery("size", "20"), "%d", &size)
	filters := map[string]string{}
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filter_") && len(values) > 0 {
			filters[strings.TrimPrefix(key, "filter_")] = values[0]
		}
	}
	return domain.PageRequest{Page: page, Size: size, Sort: c.DefaultQuery("sort", "created_at"), Filters: filters}
}
