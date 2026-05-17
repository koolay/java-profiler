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

type JVMEventBatchRequest struct {
	BatchID     string                `json:"batch_id"`
	CollectorID string                `json:"collector_id"`
	ReceivedAt  time.Time             `json:"received_at"`
	Events      []clickhouse.JVMEvent `json:"events"`
}

type JVMEventIngestor struct {
	Events    JVMEventStore
	Ingestion IngestionStore
}

type JVMEventStore interface {
	InsertJVMEvents(context.Context, []clickhouse.JVMEvent) error
	QueryJVMEvents(context.Context, clickhouse.JVMEventQuery) ([]clickhouse.JVMEvent, error)
}

func (i JVMEventIngestor) Ingest(ctx context.Context, req JVMEventBatchRequest) (IngestResult, error) {
	if req.BatchID == "" || req.CollectorID == "" {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch_id and collector_id are required"}, nil
	}
	events := append([]clickhouse.JVMEvent(nil), req.Events...)
	for index, event := range events {
		if event.BatchID != "" && event.BatchID != req.BatchID {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "JVM event batch_id conflicts with envelope batch_id"}, nil
		}
		if event.EventID == "" || event.EventType == "" || event.Target.Key() == "" || event.EventAt.IsZero() {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid JVM event"}, nil
		}
		events[index].BatchID = req.BatchID
	}
	batch := clickhouse.IngestionBatch{
		BatchID:     req.BatchID,
		CollectorID: req.CollectorID,
		BatchType:   domain.BatchTypeJVMEvent,
		ReceivedAt:  firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:      clickhouse.IngestionClaimed,
		PayloadHash: jvmEventHash(events),
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
	if err := i.Events.InsertJVMEvents(ctx, events); err != nil {
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

func jvmEventHash(events []clickhouse.JVMEvent) string {
	data, _ := json.Marshal(events)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
