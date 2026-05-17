package domain

import (
	"testing"
	"time"

	rootdomain "github.com/koolay/java-profiler/domain"
)

func TestBuildFlamegraphAggregatesStacks(t *testing.T) {
	got := BuildFlamegraph([]FlamegraphSample{
		{Frames: []string{"A", "B"}, Value: 10},
		{Frames: []string{"A", "C"}, Value: 5},
	}, 10)
	if got.Root.Value != 15 || len(got.Root.Children) != 1 || got.Root.Children[0].Value != 15 {
		t.Fatalf("unexpected flamegraph: %+v", got)
	}
}

func TestApplyProfileSemanticsAddsDisplayValues(t *testing.T) {
	got := BuildFlamegraph([]FlamegraphSample{{Frames: []string{"A"}, Value: uint64(2 * time.Second)}}, 10)
	got = ApplyProfileSemantics(got, rootdomain.ProfileTypeCPU, rootdomain.TimeWindow{StartedAt: time.Unix(0, 0), EndsAt: time.Unix(10, 0)})

	if got.Semantics.ValueUnit != "nanoseconds" || got.Root.DisplayValue != "2.00 s · 0.20 cores" {
		t.Fatalf("unexpected semantic flamegraph: %+v", got)
	}
	if got.Root.Children[0].DisplayValue != "2.00 s · 0.20 cores" {
		t.Fatalf("child display value = %q", got.Root.Children[0].DisplayValue)
	}
}

func TestBuildFlamegraphMarksPartial(t *testing.T) {
	got := BuildFlamegraph([]FlamegraphSample{{Frames: []string{"A", "B", "C"}, Value: 1}}, 2)
	if !got.Metadata.Partial || got.Metadata.OmittedNodes == 0 {
		t.Fatalf("expected partial metadata: %+v", got.Metadata)
	}
}
