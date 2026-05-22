package app

import (
	"context"
	"sort"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type ServiceSelectors struct {
	Targets []clickhouse.ProfileSelector `json:"targets"`
}

func QueryServiceSelectors(ctx context.Context, repo ProfileQueryStore, q clickhouse.ProfileQuery) (ServiceSelectors, error) {
	selectors, err := repo.QueryProfileSelectors(ctx, q)
	if err != nil {
		return ServiceSelectors{}, err
	}
	seen := map[string]clickhouse.ProfileSelector{}
	for _, selector := range selectors {
		key := selector.Namespace + "\x00" + selector.Service + "\x00" + selector.Pod
		seen[key] = selector
	}
	out := make([]clickhouse.ProfileSelector, 0, len(seen))
	for _, selector := range seen {
		out = append(out, selector)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		return out[i].Pod < out[j].Pod
	})
	return ServiceSelectors{Targets: out}, nil
}
