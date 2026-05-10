package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

func TestQueryIngestionHealthTotalsLossMetadata(t *testing.T) {
	repo := clickhouse.NewIngestionRepository()
	for _, batch := range []clickhouse.IngestionBatch{
		{
			BatchID:            "accepted",
			CollectorID:        "collector-a",
			BatchType:          domain.BatchTypeProfile,
			ReceivedAt:         time.Unix(100, 0),
			Status:             clickhouse.IngestionAccepted,
			PayloadHash:        "a",
			DroppedSampleCount: 5,
			DroppedStackCount:  2,
			Truncated:          true,
		},
		{
			BatchID:            "retryable",
			CollectorID:        "collector-a",
			BatchType:          domain.BatchTypeProfile,
			ReceivedAt:         time.Unix(101, 0),
			Status:             clickhouse.IngestionRetryable,
			Retryable:          true,
			PayloadHash:        "b",
			Message:            "storage unavailable",
			DroppedSampleCount: 3,
			DroppedStackCount:  1,
		},
	} {
		if _, err := repo.Record(context.Background(), batch); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}

	health, err := QueryIngestionHealth(context.Background(), repo)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if health.Totals.DroppedSamples != 8 {
		t.Fatalf("dropped samples = %d", health.Totals.DroppedSamples)
	}
	if health.Totals.DroppedStacks != 3 {
		t.Fatalf("dropped stacks = %d", health.Totals.DroppedStacks)
	}
	if health.Totals.TruncatedBatches != 1 {
		t.Fatalf("truncated batches = %d", health.Totals.TruncatedBatches)
	}
}
