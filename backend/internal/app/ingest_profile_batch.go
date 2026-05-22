package app

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
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
	QueryFlamegraphSamples(context.Context, clickhouse.ProfileQuery) ([]clickhouse.FlamegraphSample, error)
	QueryTopStackSamples(context.Context, clickhouse.ProfileQuery) ([]clickhouse.TopStackSample, error)
	QueryProfileTargetSummary(context.Context, clickhouse.ProfileQuery) ([]clickhouse.ProfileTargetSummary, error)
	QueryProfileSelectors(context.Context, clickhouse.ProfileQuery) ([]clickhouse.ProfileSelector, error)
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
	hasher := sha256.New()
	writeProfileBatchMetadata(hasher, metadata)
	for _, sample := range samples {
		writeProfileSample(hasher, sample)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func writeProfileBatchMetadata(h hash.Hash, metadata profiling.ProfileBatchMetadata) {
	writeInt(h, metadata.WindowRawSampleCount)
	writeInt(h, metadata.WindowAggregatedSampleCount)
	writeInt(h, metadata.BatchSampleCount)
	writeInt(h, metadata.DroppedSampleCount)
	writeInt(h, metadata.DroppedStackCount)
	writeBool(h, metadata.Truncated)
	writeInt(h, metadata.PartIndex)
	writeInt(h, metadata.PartCount)
}

func writeProfileSample(h hash.Hash, sample clickhouse.ProfileSample) {
	writeString(h, sample.Target.Cluster)
	writeString(h, sample.Target.Namespace)
	writeString(h, sample.Target.Workload)
	writeString(h, sample.Target.Pod)
	writeString(h, sample.Target.Container)
	writeString(h, sample.Target.Node)
	writeString(h, sample.Target.PodUID)
	writeInt(h, sample.Target.ProcessID)
	writeUnixNano(h, sample.Target.JVMStartTime)
	writeString(h, sample.Target.RuntimeVendor)
	writeString(h, sample.Target.RuntimeVersion)
	writeString(h, sample.Target.Service)
	writeString(h, sample.ProfileType.String())
	writeUnixNano(h, sample.StartedAt)
	writeUnixNano(h, sample.EndedAt)
	writeString(h, sample.StackID)
	writeStrings(h, sample.Frames)
	writeUint64(h, sample.Value)
	writeBool(h, sample.Truncated)
}

func writeStrings(h hash.Hash, values []string) {
	writeInt(h, len(values))
	for _, value := range values {
		writeString(h, value)
	}
}

func writeString(h hash.Hash, value string) {
	writeUint64(h, uint64(len(value)))
	_, _ = h.Write([]byte(value))
}

func writeInt(h hash.Hash, value int) {
	writeInt64(h, int64(value))
}

func writeInt64(h hash.Hash, value int64) {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutVarint(buf[:], value)
	_, _ = h.Write(buf[:n])
}

func writeUint64(h hash.Hash, value uint64) {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], value)
	_, _ = h.Write(buf[:n])
}

func writeBool(h hash.Hash, value bool) {
	if value {
		_, _ = h.Write([]byte{1})
		return
	}
	_, _ = h.Write([]byte{0})
}

func writeUnixNano(h hash.Hash, value time.Time) {
	writeInt64(h, value.UnixNano())
}

func firstNonZero(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
