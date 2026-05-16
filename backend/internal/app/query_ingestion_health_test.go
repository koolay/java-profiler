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

	health, err := QueryIngestionHealth(context.Background(), repo, nil)
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

func TestQueryIngestionHealthUsesOptimizedRepositoryPath(t *testing.T) {
	repo := &optimizedIngestionHealthStore{
		health: clickhouse.IngestionHealthReport{
			Batches: []clickhouse.IngestionHealthBatch{
				{
					BatchType:   domain.BatchTypeProfile,
					Status:      clickhouse.IngestionAccepted,
					Retryable:   false,
					Count:       2,
					LatestAt:    time.Unix(101, 0),
					LastMessage: "accepted",
				},
			},
			Totals: clickhouse.IngestionHealthTotals{
				Accepted:       2,
				DroppedSamples: 3,
			},
		},
	}

	health, err := QueryIngestionHealth(context.Background(), repo, nil)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !repo.healthQueried {
		t.Fatalf("expected optimized health query path")
	}
	if repo.listQueried {
		t.Fatalf("expected list query path to stay unused")
	}
	if len(health.Batches) != 1 {
		t.Fatalf("expected one batch, got %+v", health.Batches)
	}
	if health.Totals.Accepted != 2 || health.Totals.DroppedSamples != 3 {
		t.Fatalf("unexpected totals: %+v", health.Totals)
	}
}

type optimizedIngestionHealthStore struct {
	health        clickhouse.IngestionHealthReport
	healthQueried bool
	listQueried   bool
}

func (s *optimizedIngestionHealthStore) Record(context.Context, clickhouse.IngestionBatch) (clickhouse.IngestionStatus, error) {
	return "", nil
}

func (s *optimizedIngestionHealthStore) ListIngestionBatches(context.Context, clickhouse.IngestionQuery) ([]clickhouse.IngestionBatch, error) {
	s.listQueried = true
	return nil, nil
}

func (s *optimizedIngestionHealthStore) QueryIngestionHealth(context.Context, clickhouse.IngestionQuery) (clickhouse.IngestionHealthReport, error) {
	s.healthQueried = true
	return s.health, nil
}
