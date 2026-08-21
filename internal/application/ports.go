package application

import (
	"context"
	"io"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
	"github.com/wyw14/cry052/internal/service"
)

type Store interface {
	ApplyMutation(context.Context, persistence.Mutation) error
	CreateDataSource(context.Context, domain.DataSource) error
	GetDataSource(context.Context, string) (domain.DataSource, error)
	ListDataSources(context.Context, domain.PageRequest) (domain.Page[domain.DataSource], error)
	SaveDataSource(context.Context, domain.DataSource, int64) error
	PutTable(context.Context, domain.TableSchema) error
	GetTable(context.Context, string) (domain.TableSchema, error)
	SaveTable(context.Context, domain.TableSchema, int64) error
	ListTables(context.Context, string, domain.PageRequest) (domain.Page[domain.TableSchema], error)
	CreatePolicy(context.Context, domain.PolicyVersion) error
	GetPolicy(context.Context, string) (domain.PolicyVersion, error)
	SavePolicy(context.Context, domain.PolicyVersion, int64) error
	ListPolicies(context.Context, domain.PageRequest) (domain.Page[domain.PolicyVersion], error)
	CreatePreview(context.Context, domain.Preview) error
	GetPreview(context.Context, string) (domain.Preview, error)
	SavePreview(context.Context, domain.Preview) error
	CreateBatch(context.Context, domain.Batch) error
	GetBatch(context.Context, string) (domain.Batch, error)
	GetBatchByIdempotency(context.Context, string) (domain.Batch, error)
	SaveBatch(context.Context, domain.Batch, int64) error
	ListBatches(context.Context, domain.PageRequest) (domain.Page[domain.Batch], error)
	CreateApproval(context.Context, domain.Approval) error
	GetApproval(context.Context, string) (domain.Approval, error)
	SaveApproval(context.Context, domain.Approval, int64) error
	ListApprovals(context.Context, domain.PageRequest) (domain.Page[domain.Approval], error)
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, domain.PageRequest) (domain.Page[domain.AuditEvent], error)
}

type SampleData interface {
	Rows(context.Context, domain.TableSchema, int) ([]service.Row, error)
}

type Notifier interface {
	Notify(context.Context, string, string, map[string]string) error
}

type Redactor interface {
	Text(string) string
	Fields(map[string]string) map[string]string
}

type AttachmentStore interface {
	Save(context.Context, string, string, io.Reader) (string, error)
	Delete(context.Context, string) error
}

type CallbackSink interface {
	Deliver(context.Context, domain.LocalCallback) error
	List(context.Context) ([]domain.LocalCallback, error)
}

type Scheduler interface {
	Schedule(context.Context, domain.ScheduledEvent) error
	Cancel(context.Context, string) error
	List(context.Context) ([]domain.ScheduledEvent, error)
}

type Clock interface {
	Now() time.Time
}

type operationContextKey struct{}

type operationContextState struct {
	StartedAt time.Time
	ParentErr error
}

func detachedOperationContext(parent context.Context) context.Context {
	state := operationContextState{StartedAt: time.Now(), ParentErr: parent.Err()}
	return context.WithValue(context.WithoutCancel(parent), operationContextKey{}, state)
}
