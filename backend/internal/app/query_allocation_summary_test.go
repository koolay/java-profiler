package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/domain"
)

func TestAllocationSummaryRanksPathsSelfFramesAndInsights(t *testing.T) {
	repo := clickhouse.NewProfileRepository()
	now := time.Unix(100, 0).UTC()
	samples := []clickhouse.ProfileSample{
		{
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1"},
			ProfileType: domain.ProfileTypeAllocBytes,
			StartedAt:   now,
			EndedAt:     now.Add(time.Second),
			StackID:     "string-a",
			Frames:      []string{"root", "java/util/Arrays.copyOf:3332", "java/lang/StringBuilder.append:136"},
			Value:       8,
		},
		{
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1"},
			ProfileType: domain.ProfileTypeAllocBytes,
			StartedAt:   now,
			EndedAt:     now.Add(time.Second),
			StackID:     "db-a",
			Frames:      []string{"root", "com/acme/MultiQueryBuilder.build:107", "com/acme/BusinessDataReader.load:239"},
			Value:       2,
		},
	}
	if err := repo.InsertProfileBatch(context.Background(), "batch-1", samples); err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	got, err := QueryAllocationSummary(context.Background(), repo, nil, nil, AllocationSummaryQuery{
		Namespace:      "prod",
		Service:        "checkout",
		ProfileType:    domain.ProfileTypeAllocBytes,
		Start:          now.Add(-time.Second),
		End:            now.Add(2 * time.Second),
		PathLimit:      10,
		SelfFrameLimit: 10,
	}, nil)
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
	if !got.Coverage.HasData || got.Coverage.TotalValue != 10 {
		t.Fatalf("coverage = %+v", got.Coverage)
	}
	if len(got.TopPaths) != 2 || got.TopPaths[0].Category != "string_construction" {
		t.Fatalf("top paths = %+v", got.TopPaths)
	}
	if got.TopPaths[0].Percent != 80 {
		t.Fatalf("percent = %v", got.TopPaths[0].Percent)
	}
	if len(got.TopSelfFrames) != 2 || got.TopSelfFrames[0].Frame != "java/lang/StringBuilder.append:136" {
		t.Fatalf("self frames = %+v", got.TopSelfFrames)
	}
	if len(got.Insights) == 0 || got.Insights[0].MessageCode != "allocation.string_construction.dominant" {
		t.Fatalf("insights = %+v", got.Insights)
	}
}

func TestAllocationSummaryNormalizesScopeAndRejectsInvalidQueries(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	normalized, err := normalizeAllocationSummaryQuery(AllocationSummaryQuery{
		Namespace:      " prod ",
		Service:        "all",
		Pod:            " ",
		ProfileType:    domain.ProfileTypeAllocObjects,
		Start:          now,
		End:            now.Add(time.Minute),
		PathLimit:      999,
		SelfFrameLimit: -1,
	})
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if normalized.Namespace != "prod" || normalized.Service != "" || normalized.Pod != "" {
		t.Fatalf("scope not normalized: %+v", normalized)
	}
	if normalized.PathLimit != MaxAllocationPathLimit || normalized.SelfFrameLimit != DefaultAllocationSelfFrameLimit {
		t.Fatalf("limits not bounded: %+v", normalized)
	}

	_, err = normalizeAllocationSummaryQuery(AllocationSummaryQuery{
		Namespace:   "prod",
		ProfileType: domain.ProfileTypeCPU,
		Start:       now,
		End:         now.Add(time.Minute),
	})
	if err == nil {
		t.Fatal("expected invalid profile type error")
	}
}

func TestAllocationSummaryMarksPathAndSelfFrameLimits(t *testing.T) {
	repo := clickhouse.NewProfileRepository()
	now := time.Unix(100, 0).UTC()
	samples := []clickhouse.ProfileSample{
		{Target: domain.TargetIdentity{Namespace: "prod"}, ProfileType: domain.ProfileTypeAllocBytes, StartedAt: now, EndedAt: now, StackID: "a", Frames: []string{"root", "A.alloc"}, Value: 3},
		{Target: domain.TargetIdentity{Namespace: "prod"}, ProfileType: domain.ProfileTypeAllocBytes, StartedAt: now, EndedAt: now, StackID: "b", Frames: []string{"root", "B.alloc"}, Value: 2},
		{Target: domain.TargetIdentity{Namespace: "prod"}, ProfileType: domain.ProfileTypeAllocBytes, StartedAt: now, EndedAt: now, StackID: "c", Frames: []string{"root", "C.alloc"}, Value: 1},
	}
	if err := repo.InsertProfileBatch(context.Background(), "batch-1", samples); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	got, err := QueryAllocationSummary(context.Background(), repo, nil, nil, AllocationSummaryQuery{
		Namespace:      "prod",
		ProfileType:    domain.ProfileTypeAllocBytes,
		Start:          now.Add(-time.Second),
		End:            now.Add(time.Second),
		PathLimit:      2,
		SelfFrameLimit: 2,
	}, nil)
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
	if !got.Coverage.Partial {
		t.Fatalf("expected partial coverage: %+v", got.Coverage)
	}
	if got.Coverage.OmittedPathsLowerBound != 1 {
		t.Fatalf("omitted paths = %d", got.Coverage.OmittedPathsLowerBound)
	}
	if !hasReason(got.Coverage.PartialReasons, "path_limit") || !hasReason(got.Coverage.PartialReasons, "self_frame_limit") {
		t.Fatalf("partial reasons = %v", got.Coverage.PartialReasons)
	}
}

func TestCategorizeAllocationFrames(t *testing.T) {
	tests := map[string][]string{
		"string_construction":     {"java/lang/AbstractStringBuilder.append:448", "java/lang/StringBuilder.append:136"},
		"array_copy":              {"java/util/Arrays.copyOf:3332"},
		"collection_growth":       {"java/util/HashMap.resize:704"},
		"thread_local_cleanup":    {"kd/bos/thread/ThreadLocalUtils.getFieldValue:154"},
		"database_query_building": {"com/acme/MultiQueryBuilder.build:107", "com/acme/Repository.find:42"},
		"url_or_config_building":  {"com/acme/DataSourceURLCreatorBase.appendURL:113", "com/acme/DBConfig.fromDBInstance:416"},
		"serialization_or_json":   {"com/fasterxml/jackson/ObjectMapper.writeValueAsString"},
		"native_or_runtime":       {"libc-2.17.so.__clock_gettime"},
		"application_other":       {"com/acme/CheckoutService.handle:42"},
	}
	for want, frames := range tests {
		if got := CategorizeAllocationFrames(frames); got != want {
			t.Fatalf("category for %v = %s, want %s", frames, got, want)
		}
	}
}

func hasReason(reasons []string, reason string) bool {
	for _, item := range reasons {
		if item == reason {
			return true
		}
	}
	return false
}
