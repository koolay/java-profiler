package app

import (
	"context"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	backenddomain "github.com/koolay/java-profiler/backend/internal/domain"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/domain"
)

type FlamegraphQuery struct {
	Namespace   string
	Service     string
	Pod         string
	ProfileType domain.ProfileType
	Start       time.Time
	End         time.Time
	Limit       int
	NodeLimit   int
}

type FlamegraphQuerier struct {
	Profiles ProfileQueryStore
	Metrics  *metrics.Exporter
}

func (q FlamegraphQuerier) Query(ctx context.Context, query FlamegraphQuery) (backenddomain.FlamegraphResult, error) {
	fetchStarted := time.Now()
	samples, err := q.Profiles.QueryFlamegraphSamples(ctx, clickhouse.ProfileQuery{
		Namespace:   query.Namespace,
		Service:     query.Service,
		Pod:         query.Pod,
		ProfileType: query.ProfileType,
		Start:       query.Start,
		End:         query.End,
		Limit:       query.Limit,
	})
	if err != nil {
		return backenddomain.FlamegraphResult{}, err
	}
	recordMetric(q.Metrics, "java_profiler_query_flamegraph_fetch_seconds_total", time.Since(fetchStarted).Seconds())
	flamegraphSamples := make([]backenddomain.FlamegraphSample, 0, len(samples))
	for _, sample := range samples {
		flamegraphSamples = append(flamegraphSamples, backenddomain.FlamegraphSample{
			Frames: sample.Frames,
			Value:  sample.Value,
		})
	}
	buildStarted := time.Now()
	result := backenddomain.BuildFlamegraph(flamegraphSamples, query.NodeLimit)
	result = backenddomain.ApplyProfileSemantics(result, query.ProfileType, domain.TimeWindow{StartedAt: query.Start, EndsAt: query.End})
	recordMetric(q.Metrics, "java_profiler_query_flamegraph_build_seconds_total", time.Since(buildStarted).Seconds())
	recordMetric(q.Metrics, "java_profiler_query_flamegraph_scanned_samples_total", float64(result.Metadata.ScannedSamples))
	recordMetric(q.Metrics, "java_profiler_query_flamegraph_omitted_nodes_total", float64(result.Metadata.OmittedNodes))
	return result, nil
}
