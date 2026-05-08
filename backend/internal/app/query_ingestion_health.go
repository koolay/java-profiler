package app

import (
	"context"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
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
	Accepted  int `json:"accepted"`
	Duplicate int `json:"duplicate"`
	Retryable int `json:"retryable"`
	Rejected  int `json:"rejected"`
}

func QueryIngestionHealth(ctx context.Context, repo IngestionQueryStore) (IngestionHealth, error) {
	batches, err := repo.ListIngestionBatches(ctx, clickhouse.IngestionQuery{Limit: 1000})
	if err != nil {
		return IngestionHealth{}, err
	}
	grouped := map[string]IngestionHealthBatch{}
	var totals IngestionHealthTotals
	for _, batch := range batches {
		key := string(batch.BatchType) + "|" + string(batch.Status)
		current := grouped[key]
		current.BatchType = batch.BatchType
		current.Status = batch.Status
		current.Retryable = batch.Retryable
		current.Count++
		if batch.ReceivedAt.After(current.LatestAt) {
			current.LatestAt = batch.ReceivedAt
			current.LastMessage = batch.Message
		}
		grouped[key] = current
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
	return IngestionHealth{Batches: out, Totals: totals}, nil
}
