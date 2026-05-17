package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

func TestQueryServiceProfileSummaryRanksPodJVMTargets(t *testing.T) {
	repo := clickhouse.NewProfileRepository()
	now := time.Unix(100, 0).UTC()
	if err := repo.InsertProfileBatch(context.Background(), "batch", []clickhouse.ProfileSample{
		{Target: domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-a", Container: "app", ProcessID: 11, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), Value: uint64(2 * time.Second)},
		{Target: domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-b", Container: "app", ProcessID: 12, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), Value: uint64(8 * time.Second)},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := QueryServiceProfileSummary(context.Background(), repo, clickhouse.ProfileQuery{
		Namespace:   "prod",
		Service:     "checkout",
		ProfileType: domain.ProfileTypeCPU,
		Start:       now,
		End:         now.Add(10 * time.Second),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Targets) != 2 || got.Targets[0].Pod != "pod-b" {
		t.Fatalf("expected pod-b to rank first, got %+v", got.Targets)
	}
	if got.Targets[0].DisplayValue != "8.00 s · 0.80 cores" || got.Targets[0].PercentOfTotal != "80.0%" {
		t.Fatalf("unexpected target semantics: %+v", got.Targets[0])
	}
}
