package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

type ProfileBatchRequest struct {
	BatchID     string
	CollectorID string
	ReceivedAt  time.Time
	Samples     []clickhouse.ProfileSample
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

func (i ProfileBatchIngestor) Ingest(ctx context.Context, req ProfileBatchRequest) (IngestResult, error) {
	if req.BatchID == "" || req.CollectorID == "" {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch_id and collector_id are required"}, nil
	}
	for _, sample := range req.Samples {
		if !sample.ProfileType.IsValid() || sample.Target.Key() == "" {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid profile sample"}, nil
		}
	}
	status, err := i.Ingestion.Record(ctx, clickhouse.IngestionBatch{
		BatchID:     req.BatchID,
		CollectorID: req.CollectorID,
		BatchType:   domain.BatchTypeProfile,
		ReceivedAt:  firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:      clickhouse.IngestionAccepted,
		PayloadHash: payloadHash(req.Samples),
	})
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionDuplicate {
		return IngestResult{Status: clickhouse.IngestionDuplicate, Message: "duplicate batch ignored"}, nil
	}
	if status == clickhouse.IngestionRejected {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch id reused with different payload"}, nil
	}
	if err := i.Profiles.InsertProfileBatch(ctx, req.BatchID, req.Samples); err != nil {
		if errors.Is(err, clickhouse.ErrDuplicateBatch) {
			return IngestResult{Status: clickhouse.IngestionDuplicate, Message: "duplicate profile rows ignored"}, nil
		}
		return IngestResult{Status: clickhouse.IngestionRetryable, Retryable: true, Message: err.Error()}, nil
	}
	return IngestResult{Status: clickhouse.IngestionAccepted, Message: "accepted"}, nil
}

func payloadHash(samples []clickhouse.ProfileSample) string {
	data, _ := json.Marshal(samples)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstNonZero(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
