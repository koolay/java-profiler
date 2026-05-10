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
	snapshots := append([]clickhouse.ThreadSnapshot(nil), req.Snapshots...)
	for index, snapshot := range snapshots {
		if snapshot.BatchID != "" && snapshot.BatchID != req.BatchID {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "thread snapshot batch_id conflicts with envelope batch_id"}, nil
		}
		if snapshot.Target.Key() == "" || snapshot.SnapshotAt.IsZero() || snapshot.ThreadID == 0 {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid thread snapshot"}, nil
		}
		snapshots[index].BatchID = req.BatchID
	}
	deadlocks := append([]clickhouse.DeadlockEvent(nil), req.Deadlocks...)
	for _, event := range deadlocks {
		if event.EventID == "" || event.Target.Key() == "" || event.EventAt.IsZero() || event.CycleID == "" {
			return IngestResult{Status: clickhouse.IngestionRejected, Message: "invalid deadlock event"}, nil
		}
	}
	batch := clickhouse.IngestionBatch{
		BatchID:     req.BatchID,
		CollectorID: req.CollectorID,
		BatchType:   domain.BatchTypeThreadSnapshot,
		ReceivedAt:  firstNonZero(req.ReceivedAt, time.Now().UTC()),
		Status:      clickhouse.IngestionClaimed,
		PayloadHash: threadSnapshotHash(snapshots, deadlocks),
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
	if err := i.Threads.InsertSnapshots(ctx, snapshots, deadlocks); err != nil {
		return IngestResult{Status: clickhouse.IngestionRetryable, Retryable: true, Message: err.Error()}, nil
	}
	batch.Status = clickhouse.IngestionAccepted
	status, err = i.Ingestion.Record(ctx, batch)
	if err != nil {
		return IngestResult{}, err
	}
	if status == clickhouse.IngestionRejected {
		batch.Status = clickhouse.IngestionRejected
		batch.Message = "batch id reused with different payload"
		_, _ = i.Ingestion.Record(ctx, batch)
		return IngestResult{Status: clickhouse.IngestionRejected, Message: batch.Message}, nil
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
