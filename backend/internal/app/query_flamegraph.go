package app

import (
	"context"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	backenddomain "github.com/koolay/java-profiler/backend/internal/domain"
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
}

func (q FlamegraphQuerier) Query(ctx context.Context, query FlamegraphQuery) (backenddomain.FlamegraphResult, error) {
	samples, err := q.Profiles.QuerySamples(ctx, clickhouse.ProfileQuery{
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
	return backenddomain.BuildFlamegraph(samples, query.NodeLimit), nil
}
