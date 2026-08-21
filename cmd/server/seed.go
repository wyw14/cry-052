package main

import (
	"context"
	"errors"
	"time"

	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/postgres"
	"github.com/wyw14/cry052/internal/service"
)

func seedDemo(ctx context.Context, store *postgres.Store, samples *platform.SampleDatabase) error {
	now := time.Now().UTC()
	source, err := domain.NewDataSource("demo-source", "本地客户样例库", domain.DataSourceSample, "secret/demo-source", now)
	if err != nil {
		return err
	}
	if err := source.MarkReady(source.Version, now); err != nil {
		return err
	}
	if err := store.CreateDataSource(ctx, source); err != nil && !errors.Is(err, domain.ErrConflict) {
		return err
	}
	fields := []domain.FieldSchema{
		{Name: "customer_id", DataType: "text", Sensitivity: domain.SensitivityInternal, Category: "identity", Scopes: []string{"ops"}},
		{Name: "full_name", DataType: "text", Sensitivity: domain.SensitivityConfidential, Category: "personal_name", Scopes: []string{"masked_export"}},
		{Name: "mobile", DataType: "text", Sensitivity: domain.SensitivityRestricted, Category: "phone", Scopes: []string{"masked_export"}},
		{Name: "city", DataType: "text", Sensitivity: domain.SensitivityInternal, Category: "location", Scopes: []string{"analytics"}},
	}
	sourceTable := domain.TableSchema{ID: "demo-customers", DataSourceID: source.ID, Schema: "sample", Name: "customers", Fields: fields, Fingerprint: "demo-customers-v1", Version: 1, DiscoveredAt: now}
	targetTable := domain.TableSchema{ID: "demo-customers-masked", DataSourceID: source.ID, Schema: "sample", Name: "customers_masked", Fields: fields, Fingerprint: "demo-customers-masked-v1", Version: 1, DiscoveredAt: now}
	for _, table := range []domain.TableSchema{sourceTable, targetTable} {
		if err := store.PutTable(ctx, table); err != nil && !errors.Is(err, domain.ErrConflict) {
			return err
		}
	}
	policy := domain.PolicyVersion{ID: "demo-policy-v1", GroupID: "demo-policy-group", Name: "客户身份字段保护", Version: 1, Status: domain.PolicyApproved, Scopes: []string{"sample.customers"}, Strategies: []domain.Strategy{{Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "1", "suffix": "0"}}, {Kind: domain.StrategyMask, Parameters: map[string]string{"prefix": "3", "suffix": "4"}}, {Kind: domain.StrategyHash, Parameters: map[string]string{"secret_reference": "hash/default"}}, {Kind: domain.StrategyKeep}}, ChangeSummary: "内置离线演示策略", CreatedBy: "system-seed", CreatedAt: now, Revision: 3}
	if err := store.CreatePolicy(ctx, policy); err != nil && !errors.Is(err, domain.ErrConflict) {
		return err
	}
	samples.Seed(sourceTable.QualifiedName(), []service.Row{
		{"customer_id": "C-1001", "full_name": "张华", "mobile": "13800138000", "city": "上海"},
		{"customer_id": "C-1002", "full_name": "李敏", "mobile": "13900139000", "city": "杭州"},
		{"customer_id": "C-1003", "full_name": "王强", "mobile": "13700137000", "city": "苏州"},
	})
	samples.Seed(targetTable.QualifiedName(), nil)
	return nil
}
