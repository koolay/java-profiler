package jfr

import (
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestNormalizeMapsRequiredProfileTypes(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	startedAt := time.Unix(100, 0)
	endedAt := time.Unix(160, 0)
	samples := NormalizeWindow("batch-1", target, []Event{
		{Type: "execution_sample", Value: 1, Frames: []string{"A"}},
		{Type: "alloc_bytes", Value: 2, Frames: []string{"B"}},
		{Type: "alloc_objects", Value: 3, Frames: []string{"C"}},
		{Type: "monitor_enter", Value: 4, Frames: []string{"D"}},
		{Type: "lock_delay", Value: 5, Frames: []string{"E"}},
	}, startedAt, endedAt)
	if len(samples) != 5 {
		t.Fatalf("expected five samples, got %+v", samples)
	}
	for _, sample := range samples {
		if !sample.ProfileType.IsValid() || sample.StackID == "" || !sample.StartedAt.Equal(startedAt) || !sample.EndedAt.Equal(endedAt) {
			t.Fatalf("invalid sample: %+v", sample)
		}
	}
	byType := make(map[domain.ProfileType]uint64, len(samples))
	for _, sample := range samples {
		byType[sample.ProfileType] = sample.Value
	}
	if byType[domain.ProfileTypeCPU] != DefaultCPUExecutionSampleValueNS {
		t.Fatalf("expected CPU sample value in nanoseconds, got %+v", samples)
	}
	if byType[domain.ProfileTypeAllocBytes] != 2 ||
		byType[domain.ProfileTypeAllocObjects] != 3 ||
		byType[domain.ProfileTypeLockContention] != 4 ||
		byType[domain.ProfileTypeLockDelay] != 5 {
		t.Fatalf("non-CPU sample values should remain unchanged: %+v", samples)
	}
}

func TestNormalizeWindowAggregatesDuplicateCPUStacks(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	startedAt := time.Unix(100, 0)
	endedAt := time.Unix(160, 0)
	samples := NormalizeWindow("batch-1", target, []Event{
		{Type: "execution_sample", Value: 2, Frames: []string{"root", "Checkout.handle:42"}},
		{Type: "execution_sample", Value: 3, Frames: []string{"root", "Checkout.handle:42"}},
	}, startedAt, endedAt)

	if len(samples) != 1 {
		t.Fatalf("expected one aggregated sample, got %+v", samples)
	}
	want := uint64(5 * DefaultCPUExecutionSampleValueNS)
	if samples[0].Value != want {
		t.Fatalf("value = %d, want %d", samples[0].Value, want)
	}
}

func TestNormalizeWindowWithStatsPreservesRawCountBeforeAggregation(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	startedAt := time.Unix(100, 0)
	endedAt := time.Unix(160, 0)

	result := NormalizeWindowWithStats("batch-1", target, []Event{
		{Type: "execution_sample", Value: 2, Frames: []string{"root", "Checkout.handle:42"}},
		{Type: "execution_sample", Value: 3, Frames: []string{"root", "Checkout.handle:42"}},
		{Type: "unknown_event", Value: 100, Frames: []string{"ignored"}},
	}, startedAt, endedAt)

	if result.RawSampleCount != 2 {
		t.Fatalf("raw sample count = %d", result.RawSampleCount)
	}
	if len(result.Samples) != 1 {
		t.Fatalf("expected one aggregated sample, got %+v", result.Samples)
	}
}
