package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type HTTPConfig struct {
	Address        string
	RequestTimeout time.Duration
}

type DatabaseConfig struct {
	URL string
}

type AttachmentConfig struct {
	Directory string
	MaxBytes  int64
}

type SessionConfig struct {
	Admin, Reviewer, Auditor, Executor string
}

type Config struct {
	HTTP               HTTPConfig
	Database           DatabaseConfig
	Attachments        AttachmentConfig
	Sessions           SessionConfig
	ServiceAccountRole string
}

func Load() (Config, error) {
	timeout, err := time.ParseDuration(envOr("REQUEST_TIMEOUT", "5s"))
	if err != nil || timeout <= 0 {
		return Config{}, errors.New("REQUEST_TIMEOUT must be a positive duration")
	}
	maxUpload, err := strconv.ParseInt(envOr("MAX_UPLOAD_BYTES", "5242880"), 10, 64)
	if err != nil || maxUpload < 1024 {
		return Config{}, errors.New("MAX_UPLOAD_BYTES must be at least 1024")
	}
	cfg := Config{
		HTTP:        HTTPConfig{Address: envOr("HTTP_ADDR", ":8080"), RequestTimeout: timeout},
		Database:    DatabaseConfig{URL: envOr("DATABASE_URL", "postgres://cry052:cry052@localhost:5432/cry052?sslmode=disable")},
		Attachments: AttachmentConfig{Directory: envOr("ATTACHMENT_DIR", "./data/attachments"), MaxBytes: maxUpload},
		Sessions: SessionConfig{
			Admin:    envOr("ADMIN_SESSION_TOKEN", "local-admin-session"),
			Reviewer: envOr("REVIEWER_SESSION_TOKEN", "local-reviewer-session"),
			Auditor:  envOr("AUDITOR_SESSION_TOKEN", "local-auditor-session"),
			Executor: envOr("EXECUTOR_SESSION_TOKEN", "local-executor-session"),
		},
		ServiceAccountRole: envOr("SERVICE_ACCOUNT_ROLE", "masking_executor"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.ServiceAccountRole != "masking_executor" {
		return errors.New("service account must use the least-privilege masking_executor role")
	}
	seen := make(map[string]string, 4)
	for role, token := range map[string]string{
		"data_admin": c.Sessions.Admin, "policy_reviewer": c.Sessions.Reviewer,
		"auditor": c.Sessions.Auditor, "masking_executor": c.Sessions.Executor,
	} {
		if token == "" {
			return errors.New(role + " local session token is required")
		}
		if other, exists := seen[token]; exists {
			return errors.New(role + " local session token duplicates " + other)
		}
		seen[token] = role
	}
	return nil
}

func envOr(key, fallback string) string {
	if configured, exists := os.LookupEnv(key); exists && configured != "" {
		return configured
	}
	return fallback
}
