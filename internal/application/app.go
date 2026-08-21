package application

import (
	"time"

	"github.com/google/uuid"
	"github.com/wyw14/cry052/internal/service"
)

type App struct {
	store     Store
	samples   SampleData
	notifier  Notifier
	redactor  Redactor
	files     AttachmentStore
	callbacks CallbackSink
	scheduler Scheduler
	compiler  *service.Compiler
	preview   *service.PreviewEngine
	processor *service.BatchProcessor
	exporter  *service.ReportExporter
	workspace *WorkspaceReducer
	now       func() time.Time
	newID     func() string
}

type Dependencies struct {
	Store     Store
	Samples   SampleData
	Notifier  Notifier
	Redactor  Redactor
	Files     AttachmentStore
	Callbacks CallbackSink
	Scheduler Scheduler
	Compiler  *service.Compiler
	Preview   *service.PreviewEngine
	Processor *service.BatchProcessor
	Exporter  *service.ReportExporter
	Workspace *WorkspaceReducer
	Now       func() time.Time
	NewID     func() string
}

func New(deps Dependencies) *App {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.NewID == nil {
		deps.NewID = uuid.NewString
	}
	if deps.Workspace == nil {
		deps.Workspace = &WorkspaceReducer{}
	}
	return &App{store: deps.Store, samples: deps.Samples, notifier: deps.Notifier, redactor: deps.Redactor, files: deps.Files, callbacks: deps.Callbacks, scheduler: deps.Scheduler, compiler: deps.Compiler, preview: deps.Preview, processor: deps.Processor, exporter: deps.Exporter, workspace: deps.Workspace, now: deps.Now, newID: deps.NewID}
}

func (a *App) WorkspaceState() *WorkspaceReducer { return a.workspace }
