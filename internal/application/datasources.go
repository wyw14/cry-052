package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/persistence"
)

type CreateDataSource struct {
	Name                string
	Kind                domain.DataSourceKind
	ConnectionReference string
}

func (a *App) CreateDataSource(ctx context.Context, actor Actor, command CreateDataSource) (domain.DataSource, error) {
	ctx = detachedOperationContext(ctx)
	if err := actor.Require("data_admin"); err != nil {
		return domain.DataSource{}, err
	}
	dataSource, err := domain.NewDataSource(a.newID(), command.Name, command.Kind, command.ConnectionReference, a.now())
	if err != nil {
		return domain.DataSource{}, err
	}
	audit := a.newAuditEvent(actor, "datasource.create", "datasource/"+dataSource.ID, "success", map[string]string{"kind": string(dataSource.Kind), "connection_reference": "[redacted]"})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{CreateDataSource: &dataSource, Audit: audit}); err != nil {
		return domain.DataSource{}, fmt.Errorf("create data source: %w", err)
	}
	return dataSource, nil
}

func (a *App) ActivateDataSource(ctx context.Context, actor Actor, id string, expectedVersion int64) (domain.DataSource, error) {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin"); err != nil {
		return domain.DataSource{}, err
	}
	dataSource, err := a.store.GetDataSource(ctx, id)
	if err != nil {
		return domain.DataSource{}, err
	}
	previous := dataSource.Version
	if err := dataSource.MarkReady(expectedVersion, a.now()); err != nil {
		return domain.DataSource{}, err
	}
	audit := a.newAuditEvent(actor, "datasource.activate", "datasource/"+id, "success", nil)
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{UpdateDataSource: &persistence.Versioned[domain.DataSource]{Value: dataSource, Expected: previous}, Audit: audit}); err != nil {
		return domain.DataSource{}, err
	}
	return dataSource, nil
}

func (a *App) ListDataSources(ctx context.Context, actor Actor, request domain.PageRequest) (domain.Page[domain.DataSource], error) {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin", "auditor", "policy_reviewer"); err != nil {
		return domain.Page[domain.DataSource]{}, err
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "name": {}, "status": {}}, map[string]struct{}{"kind": {}, "status": {}})
	if err != nil {
		return domain.Page[domain.DataSource]{}, err
	}
	return a.store.ListDataSources(ctx, normalized)
}

func (a *App) RegisterTable(ctx context.Context, actor Actor, table domain.TableSchema) error {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin"); err != nil {
		return err
	}
	if err := table.Validate(); err != nil {
		return err
	}
	if _, err := a.store.GetDataSource(ctx, table.DataSourceID); err != nil {
		return fmt.Errorf("load data source: %w", err)
	}
	audit := a.newAuditEvent(actor, "catalog.register", "table/"+table.ID, "success", map[string]string{"qualified_name": table.QualifiedName()})
	return a.store.ApplyMutation(ctx, persistence.Mutation{CreateTable: &table, Audit: audit})
}

func (a *App) ListTables(ctx context.Context, actor Actor, dataSourceID string, request domain.PageRequest) (domain.Page[domain.TableSchema], error) {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin", "auditor", "policy_editor", "policy_reviewer"); err != nil {
		return domain.Page[domain.TableSchema]{}, err
	}
	if dataSourceID == "" {
		return domain.Page[domain.TableSchema]{}, domain.NewValidationError("data_source_id is required")
	}
	normalized, err := request.Normalize(map[string]struct{}{"created_at": {}, "qualified_name": {}, "discovered_at": {}, "version": {}}, map[string]struct{}{"schema": {}, "name": {}, "sensitivity": {}, "category": {}, "scope": {}})
	if err != nil {
		return domain.Page[domain.TableSchema]{}, err
	}
	return a.store.ListTables(ctx, dataSourceID, normalized)
}

func (a *App) UpdateTableClassification(ctx context.Context, actor Actor, tableID string, expectedVersion int64, fields []domain.FieldSchema) (domain.TableSchema, error) {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin", "policy_editor"); err != nil {
		return domain.TableSchema{}, err
	}
	table, err := a.store.GetTable(ctx, tableID)
	if err != nil {
		return domain.TableSchema{}, err
	}
	if table.Version != expectedVersion {
		return domain.TableSchema{}, domain.ErrVersionConflict
	}
	previous := table.Version
	table.Fields = fields
	table.Version++
	if err := table.Validate(); err != nil {
		return domain.TableSchema{}, err
	}
	audit := a.newAuditEvent(actor, "catalog.classification.update", "table/"+table.ID, "success", map[string]string{"qualified_name": table.QualifiedName()})
	if err := a.store.ApplyMutation(ctx, persistence.Mutation{UpdateTable: &persistence.Versioned[domain.TableSchema]{Value: table, Expected: previous}, Audit: audit}); err != nil {
		return domain.TableSchema{}, err
	}
	return table, nil
}
