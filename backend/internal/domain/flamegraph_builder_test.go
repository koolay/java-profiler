package domain

import (
	"testing"
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

func TestBuildFlamegraphMarksPartial(t *testing.T) {
	got := BuildFlamegraph([]FlamegraphSample{{Frames: []string{"A", "B", "C"}, Value: 1}}, 2)
	if !got.Metadata.Partial || got.Metadata.OmittedNodes == 0 {
		t.Fatalf("expected partial metadata: %+v", got.Metadata)
	}
}
