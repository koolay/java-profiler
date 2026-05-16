package clickhouse

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type IngestionStatus string

const (
	IngestionAccepted  IngestionStatus = "accepted"
	IngestionDuplicate IngestionStatus = "duplicate"
	IngestionClaimed   IngestionStatus = "claimed"
	IngestionRejected  IngestionStatus = "rejected"
	IngestionRetryable IngestionStatus = "retryable"
)

type IngestionBatch struct {
	BatchID               string
	CollectorID           string
	BatchType             domain.BatchType
	ReceivedAt            time.Time
	Status                IngestionStatus
	Retryable             bool
	PayloadHash           string
	Message               string
	RawSampleCount        int
	AggregatedSampleCount int
	BatchSampleCount      int
	DroppedSampleCount    int
	DroppedStackCount     int
	Truncated             bool
	StatusVersion         int
	RecordedAt            time.Time
}

type IngestionQuery struct {
	Limit int
}

type IngestionHealthBatch struct {
	BatchType   domain.BatchType
	Status      IngestionStatus
	Retryable   bool
	Count       int
	LatestAt    time.Time
	LastMessage string
}

type IngestionHealthTotals struct {
	Accepted         int
	Duplicate        int
	Retryable        int
	Rejected         int
	DroppedSamples   int
	DroppedStacks    int
	TruncatedBatches int
}

type IngestionHealthReport struct {
	Batches []IngestionHealthBatch
	Totals  IngestionHealthTotals
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
	prepareIngestionBatch(&batch)
	key := ingestionBatchKey(batch.BatchID, batch.BatchType)
	if existing, ok := r.batches[key]; ok {
		if existing.PayloadHash == batch.PayloadHash {
			if existing.Status == IngestionClaimed || existing.Status == IngestionRetryable {
				if batch.Status == IngestionAccepted || batch.Status == IngestionRetryable || batch.Status == IngestionRejected {
					r.batches[key] = latestIngestionBatch(existing, batch)
					return batch.Status, nil
				}
				return IngestionClaimed, nil
			}
			return IngestionDuplicate, nil
		}
		if batch.Status == IngestionRejected {
			r.batches[key] = latestIngestionBatch(existing, batch)
			return batch.Status, nil
		}
		return IngestionRejected, nil
	}
	r.batches[key] = batch
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
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RecordedAt.Equal(out[j].RecordedAt) {
			return out[i].ReceivedAt.After(out[j].ReceivedAt)
		}
		return out[i].RecordedAt.After(out[j].RecordedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *IngestionRepository) QueryIngestionHealth(ctx context.Context, q IngestionQuery) (IngestionHealthReport, error) {
	batches, err := r.ListIngestionBatches(ctx, q)
	if err != nil {
		return IngestionHealthReport{}, err
	}
	return summarizeIngestionHealthBatches(batches), nil
}

func summarizeIngestionHealthBatches(batches []IngestionBatch) IngestionHealthReport {
	grouped := map[string]IngestionHealthBatch{}
	var totals IngestionHealthTotals
	for _, batch := range batches {
		key := string(batch.BatchType) + "|" + string(batch.Status)
		current := grouped[key]
		current.BatchType = batch.BatchType
		current.Status = batch.Status
		current.Retryable = batch.Status == IngestionRetryable || batch.Retryable
		current.Count++
		if batch.ReceivedAt.After(current.LatestAt) {
			current.LatestAt = batch.ReceivedAt
			current.LastMessage = batch.Message
		}
		grouped[key] = current
		totals.DroppedSamples += batch.DroppedSampleCount
		totals.DroppedStacks += batch.DroppedStackCount
		if batch.Truncated {
			totals.TruncatedBatches++
		}
		switch batch.Status {
		case IngestionAccepted:
			totals.Accepted++
		case IngestionDuplicate:
			totals.Duplicate++
		case IngestionRetryable:
			totals.Retryable++
		case IngestionRejected:
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
	return IngestionHealthReport{Batches: out, Totals: totals}
}

func ingestionBatchKey(batchID string, batchType domain.BatchType) string {
	return batchID + "\x00" + string(batchType)
}

func prepareIngestionBatch(batch *IngestionBatch) {
	if batch.StatusVersion == 0 {
		batch.StatusVersion = StatusVersionForIngestionStatus(batch.Status)
	}
	if batch.RecordedAt.IsZero() {
		batch.RecordedAt = time.Now().UTC()
	}
}

func StatusVersionForIngestionStatus(status IngestionStatus) int {
	switch status {
	case IngestionClaimed:
		return 1
	case IngestionRetryable:
		return 2
	case IngestionAccepted:
		return 3
	case IngestionRejected:
		return 4
	case IngestionDuplicate:
		return 5
	default:
		return 0
	}
}

func latestIngestionBatch(a, b IngestionBatch) IngestionBatch {
	if b.StatusVersion != a.StatusVersion {
		if b.StatusVersion > a.StatusVersion {
			return b
		}
		return a
	}
	if !b.RecordedAt.Equal(a.RecordedAt) {
		if b.RecordedAt.After(a.RecordedAt) {
			return b
		}
		return a
	}
	if b.ReceivedAt.After(a.ReceivedAt) {
		return b
	}
	return a
}
