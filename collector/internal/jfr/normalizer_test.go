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
		{Type: "wall_clock", Value: 6, Frames: []string{"F"}},
		{Type: "io_wait", Value: 7, Frames: []string{"G"}},
	}, startedAt, endedAt)
	if len(samples) != 7 {
		t.Fatalf("expected seven samples, got %+v", samples)
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
		byType[domain.ProfileTypeLockDelay] != 5 ||
		byType[domain.ProfileTypeWallClock] != 6*DefaultWallClockSampleValueNS ||
		byType[domain.ProfileTypeIOWait] != 7*DefaultIOWaitWallSampleValueNS {
		t.Fatalf("sample values should be normalized by profile semantics: %+v", samples)
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

func TestNormalizeWindowExtractsGCJVMEvents(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	startedAt := time.Unix(100, 0)
	endedAt := time.Unix(160, 0)

	result := NormalizeWindowWithStats("batch-1", target, []Event{{
		Type:   "gc_pause",
		Value:  42_000_000,
		Frames: []string{"jdk.G1.collect"},
		Labels: map[string]string{"collector": "G1", "action": "end of minor GC", "cause": "Allocation Failure"},
	}}, startedAt, endedAt)

	if len(result.Samples) != 0 {
		t.Fatalf("GC events must not be profile samples: %+v", result.Samples)
	}
	if result.RawSampleCount != 0 || len(result.JVMEvents) != 1 {
		t.Fatalf("expected one raw JVM event, got raw=%d events=%+v", result.RawSampleCount, result.JVMEvents)
	}
	event := result.JVMEvents[0]
	if event.EventType != "gc_pause" || event.DurationNS != 42_000_000 || event.Collector != "G1" || event.Target.Pod != "checkout-1" {
		t.Fatalf("unexpected GC event: %+v", event)
	}
}
