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
	if samples[0].ProfileType != domain.ProfileTypeCPU || samples[0].Value != DefaultCPUExecutionSampleValueNS {
		t.Fatalf("expected CPU sample value in nanoseconds, got %+v", samples[0])
	}
	if samples[1].Value != 2 || samples[2].Value != 3 || samples[3].Value != 4 || samples[4].Value != 5 {
		t.Fatalf("non-CPU sample values should remain unchanged: %+v", samples)
	}
}
