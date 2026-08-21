package persistence

import "github.com/wyw14/cry052/internal/domain"

type Versioned[T any] struct {
	Value    T
	Expected int64
}

type Mutation struct {
	CreateDataSource *domain.DataSource
	UpdateDataSource *Versioned[domain.DataSource]
	CreateTable      *domain.TableSchema
	UpdateTable      *Versioned[domain.TableSchema]
	CreatePolicy     *domain.PolicyVersion
	UpdatePolicy     *Versioned[domain.PolicyVersion]
	CreateApproval   *domain.Approval
	UpdateApproval   *Versioned[domain.Approval]
	CreatePreview    *domain.Preview
	UpdatePreview    *domain.Preview
	CreateBatch      *domain.Batch
	UpdateBatch      *Versioned[domain.Batch]
	Audit            domain.AuditEvent
}
