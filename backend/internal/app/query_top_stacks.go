package app

import (
	"context"
	"sort"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type TopStack struct {
	StackID string   `json:"stack_id"`
	Frames  []string `json:"frames"`
	Value   uint64   `json:"value"`
}

func QueryTopStacks(ctx context.Context, repo ProfileQueryStore, q clickhouse.ProfileQuery) ([]TopStack, error) {
	samples, err := repo.QuerySamples(ctx, q)
	if err != nil {
		return nil, err
	}
	byStack := map[string]TopStack{}
	for _, sample := range samples {
		stack := byStack[sample.StackID]
		stack.StackID = sample.StackID
		stack.Frames = sample.Frames
		stack.Value += sample.Value
		byStack[sample.StackID] = stack
	}
	out := make([]TopStack, 0, len(byStack))
	for _, stack := range byStack {
		out = append(out, stack)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out, nil
}
