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

type limitedThreadSnapshotStore interface {
	ListSnapshotsLimited(context.Context, string, string, int) ([]clickhouse.ThreadSnapshot, error)
}

func QueryThreadDiagnosis(ctx context.Context, repo ThreadStore, namespace, service string, limit int) (ThreadDiagnosis, error) {
	limit = boundedQueryLimit(limit, DefaultThreadDiagnosisLimit, MaxThreadDiagnosisLimit)
	snapshots, err := listThreadSnapshots(ctx, repo, namespace, service, limit+1)
	if err != nil {
		return ThreadDiagnosis{}, err
	}
	partial := len(snapshots) > limit
	if partial {
		snapshots = snapshots[:limit]
	}
	return ThreadDiagnosis{
		BusyThreads: backenddomain.BuildBusyThreads(snapshots),
		SlowThreads: backenddomain.BuildSlowThreads(snapshots),
		Partial:     partial,
	}, nil
}

func listThreadSnapshots(ctx context.Context, repo ThreadStore, namespace, service string, limit int) ([]clickhouse.ThreadSnapshot, error) {
	if limited, ok := repo.(limitedThreadSnapshotStore); ok {
		return limited.ListSnapshotsLimited(ctx, namespace, service, limit)
	}
	return repo.ListSnapshots(ctx, namespace, service)
}
