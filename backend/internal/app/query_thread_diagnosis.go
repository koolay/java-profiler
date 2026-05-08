package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	backenddomain "github.com/koolay/java-profiler/backend/internal/domain"
)

type ThreadDiagnosis struct {
	BusyThreads []backenddomain.BusyThread `json:"busy_threads"`
	SlowThreads []backenddomain.SlowThread `json:"slow_threads"`
	Partial     bool                       `json:"partial"`
}

func QueryThreadDiagnosis(ctx context.Context, repo *clickhouse.ThreadRepository, namespace, service string) (ThreadDiagnosis, error) {
	snapshots, err := repo.ListSnapshots(ctx, namespace, service)
	if err != nil {
		return ThreadDiagnosis{}, err
	}
	return ThreadDiagnosis{
		BusyThreads: backenddomain.BuildBusyThreads(snapshots),
		SlowThreads: backenddomain.BuildSlowThreads(snapshots),
	}, nil
}
