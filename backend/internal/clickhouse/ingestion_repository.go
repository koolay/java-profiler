package clickhouse

import (
	"context"
	"sync"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type IngestionStatus string

const (
	IngestionAccepted  IngestionStatus = "accepted"
	IngestionDuplicate IngestionStatus = "duplicate"
	IngestionRejected  IngestionStatus = "rejected"
	IngestionRetryable IngestionStatus = "retryable"
)

type IngestionBatch struct {
	BatchID     string
	CollectorID string
	BatchType   domain.BatchType
	ReceivedAt  time.Time
	Status      IngestionStatus
	Retryable   bool
	PayloadHash string
	Message     string
}

type IngestionQuery struct {
	Limit int
}

type IngestionRepository struct {
	mu      sync.Mutex
	batches map[string]IngestionBatch
}

func NewIngestionRepository() *IngestionRepository {
	return &IngestionRepository{batches: map[string]IngestionBatch{}}
}

func (r *IngestionRepository) Record(_ context.Context, batch IngestionBatch) (IngestionStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.batches[batch.BatchID]; ok {
		if existing.PayloadHash == batch.PayloadHash {
			return IngestionDuplicate, nil
		}
		return IngestionRejected, nil
	}
	r.batches[batch.BatchID] = batch
	return batch.Status, nil
}

func (r *IngestionRepository) ListIngestionBatches(_ context.Context, q IngestionQuery) ([]IngestionBatch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	out := make([]IngestionBatch, 0, len(r.batches))
	for _, batch := range r.batches {
		out = append(out, batch)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
