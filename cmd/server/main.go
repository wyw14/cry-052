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
	"github.com/wyw14/cry052/internal/repository/postgres"
	httpapi "github.com/wyw14/cry052/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load configuration", zap.Error(err))
	}
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startupCtx, cancel := context.WithTimeout(rootCtx, cfg.HTTP.RequestTimeout)
	defer cancel()
	store, err := postgres.Open(startupCtx, cfg.Database.URL)
	if err != nil {
		logger.Fatal("open database", zap.Error(err))
	}
	defer store.Close()
	if err := store.Migrate(startupCtx); err != nil {
		logger.Fatal("apply migrations", zap.Error(err))
	}
	app := application.New(application.Dependencies{Store: store})
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
			logger.Fatal("serve http", zap.Error(err))
		}
	}()
	<-rootCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", zap.Error(err))
	}
}
