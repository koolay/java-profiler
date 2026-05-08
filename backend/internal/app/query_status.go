package app

import (
	"context"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type TargetStatusQueryStore interface {
	LatestByService(context.Context, clickhouse.TargetStatusQuery) ([]clickhouse.TargetStatus, error)
}

func QueryTargetStatus(ctx context.Context, repo TargetStatusQueryStore, namespace, service string, start, end time.Time) ([]clickhouse.TargetStatus, error) {
	return repo.LatestByService(ctx, clickhouse.TargetStatusQuery{
		Namespace: namespace,
		Service:   service,
		Start:     start,
		End:       end,
	})
}
