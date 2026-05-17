package app

import (
	"context"
	"sort"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
)

type ServiceProfileSummary struct {
	Targets []clickhouse.ProfileTargetSummary `json:"targets"`
	Partial bool                              `json:"partial"`
}

func QueryServiceProfileSummary(ctx context.Context, repo ProfileQueryStore, q clickhouse.ProfileQuery, exporter *metrics.Exporter) (ServiceProfileSummary, error) {
	started := time.Now()
	targets, err := repo.QueryProfileTargetSummary(ctx, q)
	if err != nil {
		return ServiceProfileSummary{}, err
	}
	recordMetric(exporter, "java_profiler_query_service_summary_fetch_seconds_total", time.Since(started).Seconds())
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].TotalValue != targets[j].TotalValue {
			return targets[i].TotalValue > targets[j].TotalValue
		}
		if targets[i].Pod != targets[j].Pod {
			return targets[i].Pod < targets[j].Pod
		}
		return targets[i].ProcessID < targets[j].ProcessID
	})
	limit := boundedQueryLimit(q.Limit, 100, 500)
	partial := len(targets) > limit
	if partial {
		targets = targets[:limit]
	}
	recordMetric(exporter, "java_profiler_query_service_summary_targets_total", float64(len(targets)))
	if partial {
		recordMetric(exporter, "java_profiler_query_service_summary_partial_total", 1)
	}
	return ServiceProfileSummary{Targets: targets, Partial: partial}, nil
}
