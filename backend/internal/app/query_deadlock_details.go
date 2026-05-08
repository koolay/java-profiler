package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

func QueryDeadlocks(ctx context.Context, repo *clickhouse.ThreadRepository, namespace, service string) ([]clickhouse.DeadlockEvent, error) {
	return repo.ListDeadlocks(ctx, namespace, service)
}
