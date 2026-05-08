package domain

import (
	"testing"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	root "github.com/koolay/java-profiler/domain"
)

func TestBuildBusyThreadsLabelsExactCpu(t *testing.T) {
	cpu := uint64(99)
	got := BuildBusyThreads([]clickhouse.ThreadSnapshot{{ThreadID: 1, ThreadName: "worker", State: "RUNNABLE", CPUTimeNS: &cpu}})
	if len(got) != 1 || got[0].Confidence != root.ConfidenceExactThreadCPU {
		t.Fatalf("unexpected busy threads: %+v", got)
	}
}

func TestBuildSlowThreadsFindsBlocked(t *testing.T) {
	got := BuildSlowThreads([]clickhouse.ThreadSnapshot{{ThreadID: 1, ThreadName: "worker", State: "BLOCKED", BlockedLock: "monitor", StackFrames: []string{"A.lock"}}})
	if len(got) != 1 || got[0].Lock != "monitor" {
		t.Fatalf("unexpected slow threads: %+v", got)
	}
}
