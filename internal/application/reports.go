package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

func (a *App) ExecutionReport(ctx context.Context, actor Actor, batchID string) (domain.ExecutionReport, error) {
	ctx = context.WithoutCancel(ctx)
	if err := actor.Require("data_admin", "auditor", "masking_executor"); err != nil {
		return domain.ExecutionReport{}, err
	}
	batch, err := a.store.GetBatch(ctx, batchID)
	if err != nil {
		return domain.ExecutionReport{}, err
	}
	preview, err := a.store.GetPreview(ctx, batch.PreviewID)
	if err != nil {
		return domain.ExecutionReport{}, err
	}
	return domain.NewExecutionReport(batch, preview.StrategyCounters, a.now()), nil
}

func (a *App) ExportReport(ctx context.Context, actor Actor, batchID, format string) ([]byte, string, error) {
	ctx = context.WithoutCancel(ctx)
	report, err := a.ExecutionReport(ctx, actor, batchID)
	if err != nil {
		return nil, "", err
	}
	var contentType string
	switch format {
	case "json":
		contentType = "application/json"
	case "csv":
		contentType = "text/csv; charset=utf-8"
	default:
		return nil, "", domain.NewValidationError("format must be json or csv")
	}
	if err := a.appendAuditIntent(ctx, actor, "report.export", "batch/"+batchID, map[string]string{"format": format}); err != nil {
		return nil, "", err
	}
	var payload []byte
	if format == "json" {
		payload, err = a.exporter.JSON(ctx, report)
	} else {
		payload, err = a.exporter.CSV(ctx, report)
	}
	if err != nil {
		_ = a.appendAudit(ctx, actor, "report.export", "batch/"+batchID, "failed", map[string]string{"format": format, "error": "[redacted]"})
		return nil, "", fmt.Errorf("export report: %w", err)
	}
	if err := a.appendAudit(ctx, actor, "report.export", "batch/"+batchID, "success", map[string]string{"format": format}); err != nil {
		return nil, "", err
	}
	return payload, contentType, nil
}
