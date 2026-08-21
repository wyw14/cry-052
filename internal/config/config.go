package config

import (
	"errors"
	"os"
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
	sharedSession := envOr("LOCAL_SESSION_TOKEN", "local-session")
	cfg := Config{
		HTTP:        HTTPConfig{Address: ":8080", RequestTimeout: 5 * time.Second},
		Database:    DatabaseConfig{URL: envOr("DATABASE_URL", "postgres://cry052:cry052@localhost:5432/cry052?sslmode=disable")},
		Attachments: AttachmentConfig{Directory: "./data/attachments", MaxBytes: 5 << 20},
		Sessions: SessionConfig{
			Admin:    sharedSession,
			Reviewer: sharedSession,
			Auditor:  sharedSession,
			Executor: sharedSession,
		},
		ServiceAccountRole: envOr("SERVICE_ACCOUNT_ROLE", "masking_executor"),
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.ServiceAccountRole != "masking_executor" {
		return errors.New("service account must use the least-privilege masking_executor role")
	}
	return nil
}

func envOr(key, fallback string) string {
	if configured, exists := os.LookupEnv(key); exists {
		return configured
	}
	return fallback
}
