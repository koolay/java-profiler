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

type ThreadSnapshotBatchRequest struct {
	BatchID     string
	CollectorID string
	ReceivedAt  time.Time
	Snapshots   []clickhouse.ThreadSnapshot
	Deadlocks   []clickhouse.DeadlockEvent
}

type ThreadSnapshotIngestor struct {
	Threads   ThreadStore
	Ingestion IngestionStore
}

type ThreadStore interface {
	InsertSnapshots(context.Context, []clickhouse.ThreadSnapshot, []clickhouse.DeadlockEvent) error
	ListSnapshots(context.Context, string, string) ([]clickhouse.ThreadSnapshot, error)
	ListDeadlocks(context.Context, string, string) ([]clickhouse.DeadlockEvent, error)
}

func (i ThreadSnapshotIngestor) Ingest(ctx context.Context, req ThreadSnapshotBatchRequest) (IngestResult, error) {
	if req.BatchID == "" || req.CollectorID == "" {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch_id and collector_id are required"}, nil
	}
	for _, snapshot := range req.Snapshots {
		if snapshot.Target.Key() == "" || snapshot.SnapshotAt.IsZero() || snapshot.ThreadID == 0 {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid thread snapshot"}, nil
		}
	}
	for _, event := range req.Deadlocks {
		if event.EventID == "" || event.Target.Key() == "" || event.EventAt.IsZero() || event.CycleID == "" {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid deadlock event"}, nil
		}
	}
	batch := clickhouse.IngestionBatch{
		BatchID:     req.BatchID,
		CollectorID: req.CollectorID,
		BatchType:   domain.BatchTypeThreadSnapshot,
		ReceivedAt:  firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:      clickhouse.IngestionAccepted,
		PayloadHash: threadSnapshotHash(req.Snapshots, req.Deadlocks),
	}
	if err := i.Threads.InsertSnapshots(ctx, req.Snapshots, req.Deadlocks); err != nil {
		return IngestResult{Status: clickhouse.IngestionRetryable, Retryable: true, Message: err.Error()}, nil
	}
	status, err := i.Ingestion.Record(ctx, batch)
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionRejected {
		return IngestResult{Status: clickhouse.IngestionRejected, Message: "batch id reused with different payload"}, nil
	}
	return IngestResult{Status: clickhouse.IngestionAccepted, Message: "accepted"}, nil
}

func threadSnapshotHash(snapshots []clickhouse.ThreadSnapshot, deadlocks []clickhouse.DeadlockEvent) string {
	data, _ := json.Marshal(struct {
		Snapshots []clickhouse.ThreadSnapshot
		Deadlocks []clickhouse.DeadlockEvent
	}{Snapshots: snapshots, Deadlocks: deadlocks})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
