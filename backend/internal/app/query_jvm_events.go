package app

import (
	"context"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

type JVMEventQuery struct {
	Namespace string
	Service   string
	Pod       string
	EventType string
	Start     domain.TimeWindow
	Limit     int
}

type JVMEventEvidence struct {
	Events  []clickhouse.JVMEvent `json:"events"`
	Partial bool                  `json:"partial"`
}

func QueryJVMEvents(ctx context.Context, repo JVMEventStore, q clickhouse.JVMEventQuery) (JVMEventEvidence, error) {
	limit := boundedQueryLimit(q.Limit, 500, 5000)
	q.Limit = limit + 1
	events, err := repo.QueryJVMEvents(ctx, q)
	if err != nil {
		return JVMEventEvidence{}, err
	}
	partial := len(events) > limit
	if partial {
		events = events[:limit]
	}
	return JVMEventEvidence{Events: events, Partial: partial}, nil
}
