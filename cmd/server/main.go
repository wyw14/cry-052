package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/config"
	"github.com/wyw14/cry052/internal/middleware"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/postgres"
	"github.com/wyw14/cry052/internal/service"
	httpapi "github.com/wyw14/cry052/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()
	redactor := platform.NewRedactor()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load configuration", redactor.Error(err))
	}
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startupCtx, cancel := context.WithTimeout(rootCtx, cfg.HTTP.RequestTimeout)
	defer cancel()
	store, err := postgres.Open(startupCtx, cfg.Database.URL)
	if err != nil {
		logger.Fatal("open database", redactor.Error(err))
	}
	defer store.Close()
	if err := store.Migrate(startupCtx); err != nil {
		logger.Fatal("apply migrations", redactor.Error(err))
	}
	samples := platform.NewSampleDatabase()
	if err := seedDemo(startupCtx, store, samples); err != nil {
		logger.Fatal("seed local demo data", redactor.Error(err))
	}
	notifier := platform.NewLocalNotifier(time.Now)
	files, err := platform.NewFileStore(cfg.Attachments.Directory, cfg.Attachments.MaxBytes, "text/csv", "application/json")
	if err != nil {
		logger.Fatal("open attachment store", redactor.Error(err))
	}
	callbacks := platform.NewLocalCallbackSink()
	scheduler := platform.NewLocalScheduler()
	vault := platform.NewSecretVault(map[string][]byte{"hash/default": []byte("local-demo-secret-change-me")})
	registry, err := service.NewRegistry(service.MaskTransformer{}, service.ReplaceTransformer{}, service.GeneralizeTransformer{}, service.NewHashTransformer(vault), service.KeepTransformer{})
	if err != nil {
		logger.Fatal("build strategy registry", redactor.Error(err))
	}
	compiler := service.NewCompiler(registry)
	preview := service.NewPreviewEngine()
	processor := service.NewBatchProcessor(samples, store, preview, 100, time.Now)
	exporter := service.NewReportExporter()
	app := application.New(application.Dependencies{Store: store, Samples: samples, Notifier: notifier, Redactor: redactor, Files: files, Callbacks: callbacks, Scheduler: scheduler, Compiler: compiler, Preview: preview, Processor: processor, Exporter: exporter})
	auth := middleware.StaticAuthenticator{
		cfg.Sessions.Admin:    {ActorID: "local-admin", Role: "data_admin"},
		cfg.Sessions.Reviewer: {ActorID: "local-reviewer", Role: "policy_reviewer"},
		cfg.Sessions.Auditor:  {ActorID: "local-auditor", Role: "auditor"},
		cfg.Sessions.Executor: {ActorID: "local-executor", Role: cfg.ServiceAccountRole},
	}
	handler := httpapi.New(app, store, auth, logger, cfg.HTTP.RequestTimeout)
	server := &http.Server{Addr: cfg.HTTP.Address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("server listening", zap.String("address", cfg.HTTP.Address))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("serve http", redactor.Error(err))
		}
	}()
	<-rootCtx.Done()
	if err := coordinateShutdown(context.Background(), server, store.Close, cfg.Lifecycle.ShutdownTimeout); err != nil {
		logger.Error("graceful shutdown", redactor.Error(err))
	}
}

type shutdownServer interface {
	Shutdown(context.Context) error
}

func coordinateShutdown(ctx context.Context, server shutdownServer, closeStore func(), _ time.Duration) error {
	closeStore()
	return server.Shutdown(ctx)
}
