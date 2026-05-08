package jfr

import (
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestNormalizeMapsRequiredProfileTypes(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	samples := Normalize("batch-1", target, []Event{
		{Type: "execution_sample", Value: 1, Frames: []string{"A"}},
		{Type: "alloc_bytes", Value: 2, Frames: []string{"B"}},
		{Type: "alloc_objects", Value: 3, Frames: []string{"C"}},
		{Type: "monitor_enter", Value: 4, Frames: []string{"D"}},
		{Type: "lock_delay", Value: 5, Frames: []string{"E"}},
	})
	if len(samples) != 5 {
		t.Fatalf("expected five samples, got %+v", samples)
	}
	for _, sample := range samples {
		if !sample.ProfileType.IsValid() || sample.StackID == "" {
			t.Fatalf("invalid sample: %+v", sample)
		}
	}
}
