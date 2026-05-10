package pipeline

import (
	"testing"

	"github.com/koolay/java-profiler/contracts/profiling"
)

func TestBoundProfileSamplesMarksTruncation(t *testing.T) {
	samples := []profiling.ProfileSample{
		{StackID: "a", Value: 1},
		{StackID: "b", Value: 1},
		{StackID: "c", Value: 1},
	}
	got, meta := BoundProfileSamples(samples, 10, ProfileBatchLimits{MaxSamplesPerWindow: 2, MaxSamplesPerBatch: 1})
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if meta.WindowRawSampleCount != 10 {
		t.Fatalf("raw sample count = %d", meta.WindowRawSampleCount)
	}
	if meta.WindowAggregatedSampleCount != 3 {
		t.Fatalf("aggregated sample count = %d", meta.WindowAggregatedSampleCount)
	}
	if !meta.Truncated {
		t.Fatalf("expected truncated metadata")
	}
	if meta.DroppedSampleCount != 1 {
		t.Fatalf("dropped sample count = %d", meta.DroppedSampleCount)
	}
}

func TestBoundProfileSamplesPreservesUnderLimitAndSetsCounts(t *testing.T) {
	samples := []profiling.ProfileSample{
		{StackID: "a", Value: 1},
		{StackID: "b", Value: 1},
	}

	got, meta := BoundProfileSamples(samples, 5, ProfileBatchLimits{MaxSamplesPerWindow: 3, MaxSamplesPerBatch: 1})

	if len(got) != len(samples) {
		t.Fatalf("len = %d", len(got))
	}
	if meta.WindowRawSampleCount != 5 {
		t.Fatalf("raw sample count = %d", meta.WindowRawSampleCount)
	}
	if meta.WindowAggregatedSampleCount != len(samples) {
		t.Fatalf("aggregated sample count = %d", meta.WindowAggregatedSampleCount)
	}
	if meta.Truncated {
		t.Fatalf("metadata should not mark under-limit samples as truncated")
	}
	if meta.DroppedSampleCount != 0 || meta.DroppedStackCount != 0 {
		t.Fatalf("unexpected dropped counts: %#v", meta)
	}
}

func TestBatchMetadataForPartDoesNotMutateBase(t *testing.T) {
	base := profiling.ProfileBatchMetadata{
		WindowRawSampleCount:        100,
		WindowAggregatedSampleCount: 80,
		DroppedSampleCount:          20,
		DroppedStackCount:           10,
		Truncated:                   true,
	}

	got := BatchMetadataForPart(base, 2, 3, 40)

	if got.PartIndex != 2 {
		t.Fatalf("part index = %d", got.PartIndex)
	}
	if got.PartCount != 3 {
		t.Fatalf("part count = %d", got.PartCount)
	}
	if got.BatchSampleCount != 40 {
		t.Fatalf("batch sample count = %d", got.BatchSampleCount)
	}
	if base.PartIndex != 0 || base.PartCount != 0 || base.BatchSampleCount != 0 {
		t.Fatalf("base metadata was mutated: %#v", base)
	}
}

func TestBatchMetadataForPartCarriesWindowCountsOnlyOnFirstPart(t *testing.T) {
	base := profiling.ProfileBatchMetadata{
		WindowRawSampleCount:        100,
		WindowAggregatedSampleCount: 80,
		DroppedSampleCount:          20,
		DroppedStackCount:           10,
		Truncated:                   true,
	}

	first := BatchMetadataForPart(base, 1, 2, 50)
	if first.WindowRawSampleCount != 100 ||
		first.WindowAggregatedSampleCount != 80 ||
		first.DroppedSampleCount != 20 ||
		first.DroppedStackCount != 10 ||
		!first.Truncated {
		t.Fatalf("first part lost window metadata: %#v", first)
	}

	second := BatchMetadataForPart(base, 2, 2, 30)
	if second.WindowRawSampleCount != 0 ||
		second.WindowAggregatedSampleCount != 0 ||
		second.DroppedSampleCount != 0 ||
		second.DroppedStackCount != 0 ||
		second.Truncated {
		t.Fatalf("second part should not repeat window metadata: %#v", second)
	}
	if second.PartIndex != 2 || second.PartCount != 2 || second.BatchSampleCount != 30 {
		t.Fatalf("second part metadata = %#v", second)
	}
}
