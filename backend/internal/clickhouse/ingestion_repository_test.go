package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestIngestionRepositoryReturnsOnlyFinalBatchState(t *testing.T) {
	repo := NewIngestionRepository()
	claimed := IngestionBatch{
		BatchID:       "batch-1",
		CollectorID:   "collector-a",
		BatchType:     domain.BatchTypeProfile,
		ReceivedAt:    time.Unix(100, 0),
		Status:        IngestionClaimed,
		PayloadHash:   "hash",
		RecordedAt:    time.Unix(200, 0),
		StatusVersion: 1,
	}
	if status, err := repo.Record(context.Background(), claimed); err != nil || status != IngestionClaimed {
		t.Fatalf("record claimed = %s err=%v", status, err)
	}
	retryable := claimed
	retryable.Status = IngestionRetryable
	retryable.Retryable = true
	retryable.Message = "storage unavailable"
	retryable.RecordedAt = time.Unix(201, 0)
	if status, err := repo.Record(context.Background(), retryable); err != nil || status != IngestionRetryable {
		t.Fatalf("record retryable = %s err=%v", status, err)
	}
	accepted := claimed
	accepted.Status = IngestionAccepted
	accepted.RecordedAt = time.Unix(202, 0)
	if status, err := repo.Record(context.Background(), accepted); err != nil || status != IngestionAccepted {
		t.Fatalf("record accepted = %s err=%v", status, err)
	}

	batches, err := repo.ListIngestionBatches(context.Background(), IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected one final state, got %+v", batches)
	}
	if batches[0].Status != IngestionAccepted {
		t.Fatalf("expected accepted final state, got %+v", batches[0])
	}
}

func TestIngestionRepositoryConflictRejectionBecomesFinalState(t *testing.T) {
	repo := NewIngestionRepository()
	accepted := IngestionBatch{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		BatchType:   domain.BatchTypeProfile,
		ReceivedAt:  time.Unix(100, 0),
		Status:      IngestionAccepted,
		PayloadHash: "hash-a",
		Message:     "accepted",
		RecordedAt:  time.Unix(200, 0),
	}
	if status, err := repo.Record(context.Background(), accepted); err != nil || status != IngestionAccepted {
		t.Fatalf("record accepted = %s err=%v", status, err)
	}
	rejected := accepted
	rejected.Status = IngestionRejected
	rejected.PayloadHash = "hash-b"
	rejected.Message = "batch id reused with different payload"
	rejected.RecordedAt = time.Unix(201, 0)
	if status, err := repo.Record(context.Background(), rejected); err != nil || status != IngestionRejected {
		t.Fatalf("record rejected = %s err=%v", status, err)
	}

	batches, err := repo.ListIngestionBatches(context.Background(), IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected one final state, got %+v", batches)
	}
	if batches[0].Status != IngestionRejected {
		t.Fatalf("expected rejected final state, got %+v", batches[0])
	}
	if batches[0].Message != "batch id reused with different payload" {
		t.Fatalf("expected rejection message, got %+v", batches[0])
	}
}
