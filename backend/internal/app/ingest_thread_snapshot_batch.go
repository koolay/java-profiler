package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type ThreadSnapshotIngestor struct {
	Threads *clickhouse.ThreadRepository
}

func (i ThreadSnapshotIngestor) Ingest(ctx context.Context, snapshots []clickhouse.ThreadSnapshot, deadlocks []clickhouse.DeadlockEvent) error {
	return i.Threads.InsertSnapshots(ctx, snapshots, deadlocks)
}
