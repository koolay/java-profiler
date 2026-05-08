package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

func QueryTargetStatus(ctx context.Context, repo *clickhouse.StatusRepository, namespace, service string) ([]clickhouse.TargetStatus, error) {
	return repo.LatestByService(ctx, namespace, service)
}
