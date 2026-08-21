package domain

import (
	"net/url"
	"strings"
	"time"
)

type DataSourceKind string

const (
	DataSourcePostgres DataSourceKind = "postgres"
	DataSourceSample   DataSourceKind = "local_sample"
)

type DataSourceStatus string

const (
	DataSourceDraft  DataSourceStatus = "draft"
	DataSourceReady  DataSourceStatus = "ready"
	DataSourcePaused DataSourceStatus = "paused"
)

type DataSource struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	Kind                DataSourceKind   `json:"kind"`
	ConnectionReference string           `json:"connection_reference"`
	Status              DataSourceStatus `json:"status"`
	Version             int64            `json:"version"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

func NewDataSource(id, name string, kind DataSourceKind, reference string, now time.Time) (DataSource, error) {
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return DataSource{}, NewValidationError("data source is invalid", FieldViolation{Field: "name", Message: "name is required"})
	}
	if kind != DataSourcePostgres && kind != DataSourceSample {
		return DataSource{}, NewValidationError("data source is invalid", FieldViolation{Field: "kind", Message: "unsupported kind"})
	}
	reference, err := normalizeConnectionReference(reference)
	if err != nil {
		return DataSource{}, err
	}
	return DataSource{ID: id, Name: name, Kind: kind, ConnectionReference: reference, Status: DataSourceDraft, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func (d *DataSource) MarkReady(expected int64, now time.Time) error {
	if d.Version != expected {
		return ErrVersionConflict
	}
	if d.Status != DataSourceDraft && d.Status != DataSourcePaused {
		return ErrInvalidTransition
	}
	d.Status = DataSourceReady
	d.Version++
	d.UpdatedAt = now
	return nil
}

func RedactDSN(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return "[redacted-connection]"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	u.RawQuery = ""
	return u.String()
}
