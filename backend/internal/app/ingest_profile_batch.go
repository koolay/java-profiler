package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

type ProfileBatchRequest struct {
	BatchID     string                         `json:"batch_id"`
	CollectorID string                         `json:"collector_id"`
	ReceivedAt  time.Time                      `json:"received_at"`
	Metadata    profiling.ProfileBatchMetadata `json:"metadata"`
	Samples     []clickhouse.ProfileSample     `json:"samples"`
}

type IngestResult struct {
	Status    clickhouse.IngestionStatus
	Retryable bool
	Message   string
}

type ProfileBatchIngestor struct {
	Profiles  ProfileStore
	Ingestion IngestionStore
}

type ProfileStore interface {
	InsertProfileBatch(context.Context, string, []clickhouse.ProfileSample) error
}

type ProfileQueryStore interface {
	ProfileStore
	QuerySamples(context.Context, clickhouse.ProfileQuery) ([]clickhouse.ProfileSample, error)
}

type IngestionStore interface {
	Record(context.Context, clickhouse.IngestionBatch) (clickhouse.IngestionStatus, error)
}

type IngestionQueryStore interface {
	IngestionStore
	ListIngestionBatches(context.Context, clickhouse.IngestionQuery) ([]clickhouse.IngestionBatch, error)
}

func (i ProfileBatchIngestor) Ingest(ctx context.Context, req ProfileBatchRequest) (IngestResult, error) {
	if req.BatchID == "" || req.CollectorID == "" {
		// Without batch and collector identifiers there is no stable key to attribute ingestion health to.
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch_id and collector_id are required"}, nil
	}
	batch := clickhouse.IngestionBatch{
		BatchID:               req.BatchID,
		CollectorID:           req.CollectorID,
		BatchType:             domain.BatchTypeProfile,
		ReceivedAt:            firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:                clickhouse.IngestionClaimed,
		PayloadHash:           payloadHash(req.Samples, req.Metadata),
		RawSampleCount:        req.Metadata.WindowRawSampleCount,
		AggregatedSampleCount: req.Metadata.WindowAggregatedSampleCount,
		BatchSampleCount:      req.Metadata.BatchSampleCount,
		DroppedSampleCount:    req.Metadata.DroppedSampleCount,
		DroppedStackCount:     req.Metadata.DroppedStackCount,
		Truncated:             req.Metadata.Truncated,
	}
	for _, sample := range req.Samples {
		if !sample.ProfileType.IsValid() || sample.Target.Key() == "" {
			batch.Status = clickhouse.IngestionRejected
			batch.Message = "invalid profile sample"
			_, _ = i.Ingestion.Record(ctx, batch)
			return IngestResult{Status: clickhouse.IngestionRejected, Message: batch.Message}, nil
		}
		if sample.StartedAt.IsZero() || sample.EndedAt.IsZero() || sample.EndedAt.Before(sample.StartedAt) {
			batch.Status = clickhouse.IngestionRejected
			batch.Message = "profile sample time range is required"
			_, _ = i.Ingestion.Record(ctx, batch)
			return IngestResult{Status: clickhouse.IngestionRejected, Message: batch.Message}, nil
		}
	}
	status, err := i.Ingestion.Record(ctx, batch)
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionRejected {
		batch.Status = clickhouse.IngestionRejected
		batch.Message = "batch id reused with different payload"
		_, _ = i.Ingestion.Record(ctx, batch)
		return IngestResult{Status: clickhouse.IngestionRejected, Message: batch.Message}, nil
	}
	if status == clickhouse.IngestionDuplicate {
		return IngestResult{Status: clickhouse.IngestionDuplicate, Message: "duplicate batch ignored"}, nil
	}
	if err := i.Profiles.InsertProfileBatch(ctx, req.BatchID, req.Samples); err != nil {
		batch.Status = clickhouse.IngestionRetryable
		batch.Retryable = true
		batch.Message = err.Error()
		if _, recordErr := i.Ingestion.Record(ctx, batch); recordErr != nil {
			return IngestResult{}, recordErr
		}
		return IngestResult{Status: clickhouse.IngestionRetryable, Retryable: true, Message: err.Error()}, nil
	}
	batch.Status = clickhouse.IngestionAccepted
	status, err = i.Ingestion.Record(ctx, batch)
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionRejected {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch id reused with different payload"}, nil
	}
	return IngestResult{Status: clickhouse.IngestionAccepted, Message: "accepted"}, nil
}

func payloadHash(samples []clickhouse.ProfileSample, metadata profiling.ProfileBatchMetadata) string {
	data, _ := json.Marshal(struct {
		Samples  []clickhouse.ProfileSample     `json:"samples"`
		Metadata profiling.ProfileBatchMetadata `json:"metadata"`
	}{Samples: samples, Metadata: metadata})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstNonZero(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
