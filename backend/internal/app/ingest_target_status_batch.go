package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type TargetStatusIngestor struct {
	Statuses *clickhouse.StatusRepository
}

func (i TargetStatusIngestor) Ingest(ctx context.Context, statuses []clickhouse.TargetStatus) error {
	return i.Statuses.InsertStatuses(ctx, statuses)
}
