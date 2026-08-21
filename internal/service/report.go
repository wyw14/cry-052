package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/wyw14/cry052/internal/domain"
)

type ReportExporter struct{}

func NewReportExporter() *ReportExporter { return &ReportExporter{} }

func (e *ReportExporter) JSON(ctx context.Context, report domain.ExecutionReport) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(report, "", "  ")
}

func (e *ReportExporter) CSV(ctx context.Context, report domain.ExecutionReport) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	rows := [][]string{
		{"metric", "value"},
		{"batch_id", report.BatchID},
		{"source_rows", strconv.FormatInt(report.SourceRows, 10)},
		{"target_rows", strconv.FormatInt(report.TargetRows, 10)},
		{"rejected_rows", strconv.FormatInt(report.RejectedRows, 10)},
		{"difference", strconv.FormatInt(report.Difference, 10)},
	}
	keys := make([]string, 0, len(report.StrategyCounters))
	for key := range report.StrategyCounters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows = append(rows, []string{"strategy_" + key, strconv.Itoa(report.StrategyCounters[key])})
	}
	if err := writer.WriteAll(rows); err != nil {
		return nil, fmt.Errorf("write report csv: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
