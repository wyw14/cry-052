package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
)

type createDataSourceRequest struct {
	Name                string                `json:"name" validate:"required,min=2,max=100"`
	Kind                domain.DataSourceKind `json:"kind" validate:"required,oneof=postgres local_sample"`
	ConnectionReference string                `json:"connection_reference" validate:"required,max=200"`
}
type registerTableRequest struct {
	ID           string               `json:"id" validate:"required"`
	DataSourceID string               `json:"data_source_id" validate:"required"`
	Schema       string               `json:"schema" validate:"required"`
	Name         string               `json:"name" validate:"required"`
	Fields       []domain.FieldSchema `json:"fields" validate:"required,min=1,dive"`
	Fingerprint  string               `json:"fingerprint" validate:"required"`
	Version      int64                `json:"version" validate:"gte=1"`
}
type versionedDataSourceRequest struct {
	Version int64 `json:"version" validate:"gte=1"`
}
type classifyTableRequest struct {
	Version int64                `json:"version" validate:"gte=1"`
	Fields  []domain.FieldSchema `json:"fields" validate:"required,min=1,dive"`
}

func (s *Server) registerDataSources(group *gin.RouterGroup) {
	sources := group.Group("/data-sources")
	sources.POST("", func(c *gin.Context) {
		var input createDataSourceRequest
		if !s.bindAndValidate(c, &input) {
			return
		}
		result, err := s.app.CreateDataSource(c.Request.Context(), middleware.CurrentActor(c), application.CreateDataSource{Name: input.Name, Kind: input.Kind, ConnectionReference: input.ConnectionReference})
		s.respond(c, http.StatusCreated, result, err)
	})
	sources.GET("", func(c *gin.Context) {
		result, err := s.app.ListDataSources(c.Request.Context(), middleware.CurrentActor(c), pageRequest(c))
		s.respond(c, http.StatusOK, result, err)
	})
	sources.POST("/:id/activate", func(c *gin.Context) {
		var input versionedDataSourceRequest
		if s.bindAndValidate(c, &input) {
			result, err := s.app.ActivateDataSource(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), input.Version)
			s.respond(c, http.StatusOK, result, err)
		}
	})

	tables := group.Group("/tables")
	tables.POST("", func(c *gin.Context) {
		var input registerTableRequest
		if !s.bindAndValidate(c, &input) {
			return
		}
		table := domain.TableSchema{ID: input.ID, DataSourceID: input.DataSourceID, Schema: input.Schema, Name: input.Name, Fields: input.Fields, Fingerprint: input.Fingerprint, Version: input.Version, DiscoveredAt: time.Now()}
		err := s.app.RegisterTable(c.Request.Context(), middleware.CurrentActor(c), table)
		s.respond(c, http.StatusCreated, table, err)
	})
	tables.GET("", func(c *gin.Context) {
		result, err := s.app.ListTables(c.Request.Context(), middleware.CurrentActor(c), c.Query("data_source_id"), pageRequest(c))
		s.respond(c, http.StatusOK, result, err)
	})
	tables.PATCH("/:id/classification", func(c *gin.Context) {
		var input classifyTableRequest
		if s.bindAndValidate(c, &input) {
			result, err := s.app.UpdateTableClassification(c.Request.Context(), middleware.CurrentActor(c), c.Param("id"), input.Version, input.Fields)
			s.respond(c, http.StatusOK, result, err)
		}
	})
}
