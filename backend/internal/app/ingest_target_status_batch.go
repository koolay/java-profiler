package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

type TargetStatusBatchRequest struct {
	BatchID     string
	CollectorID string
	ReceivedAt  time.Time
	Statuses    []clickhouse.TargetStatus
}

type TargetStatusIngestor struct {
	Statuses  TargetStatusStore
	Ingestion IngestionStore
}

type TargetStatusStore interface {
	InsertStatuses(context.Context, []clickhouse.TargetStatus) error
}

func (i TargetStatusIngestor) Ingest(ctx context.Context, req TargetStatusBatchRequest) (IngestResult, error) {
	if req.BatchID == "" || req.CollectorID == "" {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch_id and collector_id are required"}, nil
	}
	statuses := append([]clickhouse.TargetStatus(nil), req.Statuses...)
	for index, status := range statuses {
		if status.BatchID != "" && status.BatchID != req.BatchID {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "status batch_id conflicts with envelope batch_id"}, nil
		}
		if status.Target.Key() == "" || status.StatusAt.IsZero() || !status.DesiredStateIsValid() || !status.Reason.IsValid() {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid target status"}, nil
		}
		statuses[index].BatchID = req.BatchID
	}
	batch := clickhouse.IngestionBatch{
		BatchID:     req.BatchID,
		CollectorID: req.CollectorID,
		BatchType:   domain.BatchTypeTargetStatus,
		ReceivedAt:  firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:      clickhouse.IngestionClaimed,
		PayloadHash: targetStatusHash(statuses),
	}
	status, err := i.Ingestion.Record(ctx, batch)
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionRejected {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch id reused with different payload"}, nil
	}
	if status == clickhouse.IngestionDuplicate {
		return IngestResult{Status: clickhouse.IngestionDuplicate, Message: "duplicate batch ignored"}, nil
	}
	if err := i.Statuses.InsertStatuses(ctx, statuses); err != nil {
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

func targetStatusHash(statuses []clickhouse.TargetStatus) string {
	data, _ := json.Marshal(statuses)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
