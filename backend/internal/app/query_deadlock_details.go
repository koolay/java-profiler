package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type limitedDeadlockStore interface {
	ListDeadlocksLimited(context.Context, string, string, int) ([]clickhouse.DeadlockEvent, error)
}

func QueryDeadlocks(ctx context.Context, repo ThreadStore, namespace, service string, limit int) ([]clickhouse.DeadlockEvent, error) {
	limit = boundedQueryLimit(limit, DefaultDeadlockLimit, MaxDeadlockLimit)
	events, err := listDeadlocks(ctx, repo, namespace, service, limit)
	if err != nil {
		return nil, err
	}
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func listDeadlocks(ctx context.Context, repo ThreadStore, namespace, service string, limit int) ([]clickhouse.DeadlockEvent, error) {
	if limited, ok := repo.(limitedDeadlockStore); ok {
		return limited.ListDeadlocksLimited(ctx, namespace, service, limit)
	}
	return repo.ListDeadlocks(ctx, namespace, service)
}
