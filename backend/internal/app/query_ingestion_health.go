package app

import (
	"context"
	"sort"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/domain"
)

type IngestionHealth struct {
	Batches []IngestionHealthBatch `json:"batches"`
	Totals  IngestionHealthTotals  `json:"totals"`
	Partial bool                   `json:"partial"`
}

type IngestionHealthBatch struct {
	BatchType   domain.BatchType           `json:"batch_type"`
	Status      clickhouse.IngestionStatus `json:"status"`
	Retryable   bool                       `json:"retryable"`
	Count       int                        `json:"count"`
	LatestAt    time.Time                  `json:"latest_at"`
	LastMessage string                     `json:"last_message"`
}

type IngestionHealthTotals struct {
	Accepted         int `json:"accepted"`
	Duplicate        int `json:"duplicate"`
	Retryable        int `json:"retryable"`
	Rejected         int `json:"rejected"`
	DroppedSamples   int `json:"dropped_samples"`
	DroppedStacks    int `json:"dropped_stacks"`
	TruncatedBatches int `json:"truncated_batches"`
}

type IngestionHealthQueryStore interface {
	IngestionQueryStore
	QueryIngestionHealth(context.Context, clickhouse.IngestionQuery) (clickhouse.IngestionHealthReport, error)
}

func QueryIngestionHealth(ctx context.Context, repo IngestionQueryStore, exporter *metrics.Exporter) (IngestionHealth, error) {
	if healthRepo, ok := repo.(IngestionHealthQueryStore); ok {
		fetchStarted := time.Now()
		report, err := healthRepo.QueryIngestionHealth(ctx, clickhouse.IngestionQuery{Limit: 1000})
		if err != nil {
			return IngestionHealth{}, err
		}
		recordMetric(exporter, "java_profiler_query_ingestion_health_fetch_seconds_total", time.Since(fetchStarted).Seconds())
		result := fromClickhouseIngestionHealth(report)
		recordMetric(exporter, "java_profiler_query_ingestion_health_batches_total", float64(len(result.Batches)))
		return result, nil
	}
	fetchStarted := time.Now()
	batches, err := repo.ListIngestionBatches(ctx, clickhouse.IngestionQuery{Limit: 1000})
	if err != nil {
		return IngestionHealth{}, err
	}
	recordMetric(exporter, "java_profiler_query_ingestion_health_fetch_seconds_total", time.Since(fetchStarted).Seconds())
	summarizeStarted := time.Now()
	result := summarizeIngestionHealth(batches)
	recordMetric(exporter, "java_profiler_query_ingestion_health_summarize_seconds_total", time.Since(summarizeStarted).Seconds())
	recordMetric(exporter, "java_profiler_query_ingestion_health_batches_total", float64(len(result.Batches)))
	return result, nil
}

func fromClickhouseIngestionHealth(report clickhouse.IngestionHealthReport) IngestionHealth {
	out := make([]IngestionHealthBatch, 0, len(report.Batches))
	for _, batch := range report.Batches {
		out = append(out, IngestionHealthBatch{
			BatchType:   batch.BatchType,
			Status:      batch.Status,
			Retryable:   batch.Retryable,
			Count:       batch.Count,
			LatestAt:    batch.LatestAt,
			LastMessage: batch.LastMessage,
		})
	}
	return IngestionHealth{
		Batches: out,
		Totals: IngestionHealthTotals{
			Accepted:         report.Totals.Accepted,
			Duplicate:        report.Totals.Duplicate,
			Retryable:        report.Totals.Retryable,
			Rejected:         report.Totals.Rejected,
			DroppedSamples:   report.Totals.DroppedSamples,
			DroppedStacks:    report.Totals.DroppedStacks,
			TruncatedBatches: report.Totals.TruncatedBatches,
		},
	}
}

func summarizeIngestionHealth(batches []clickhouse.IngestionBatch) IngestionHealth {
	grouped := map[string]IngestionHealthBatch{}
	var totals IngestionHealthTotals
	for _, batch := range batches {
		key := string(batch.BatchType) + "|" + string(batch.Status)
		current := grouped[key]
		current.BatchType = batch.BatchType
		current.Status = batch.Status
		current.Retryable = batch.Status == clickhouse.IngestionRetryable || batch.Retryable
		current.Count++
		if batch.RecordedAt.After(current.LatestAt) {
			current.LatestAt = batch.RecordedAt
			current.LastMessage = batch.Message
		}
		grouped[key] = current
		totals.DroppedSamples += batch.DroppedSampleCount
		totals.DroppedStacks += batch.DroppedStackCount
		if batch.Truncated {
			totals.TruncatedBatches++
		}
		switch batch.Status {
		case clickhouse.IngestionAccepted:
			totals.Accepted++
		case clickhouse.IngestionDuplicate:
			totals.Duplicate++
		case clickhouse.IngestionRetryable:
			totals.Retryable++
		case clickhouse.IngestionRejected:
			totals.Rejected++
		}
	}
	out := make([]IngestionHealthBatch, 0, len(grouped))
	for _, item := range grouped {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].LatestAt.Equal(out[j].LatestAt) {
			if out[i].BatchType == out[j].BatchType {
				return out[i].Status < out[j].Status
			}
			return out[i].BatchType < out[j].BatchType
		}
		return out[i].LatestAt.After(out[j].LatestAt)
	})
	return IngestionHealth{Batches: out, Totals: totals}
}
