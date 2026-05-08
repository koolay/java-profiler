package threads

import (
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestNormalizeSnapshotCreatesDeadlockEvent(t *testing.T) {
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)}
	raw := RawSnapshot{
		CapturedAtMillis:       1000,
		ThreadCPUTimeSupported: true,
		DeadlockedThreadIDs:    []int64{7},
		Threads:                []RawThread{{ID: 7, Name: "blocked-worker", State: "BLOCKED", CPUTimeNanos: 10, Stack: []string{"A.lock"}}},
	}
	snapshots, deadlocks := NormalizeSnapshot("batch-1", target, raw)
	if len(snapshots) != 1 || snapshots[0].DeadlockCycleID == "" {
		t.Fatalf("expected deadlock snapshot, got %+v", snapshots)
	}
	if len(deadlocks) != 1 || deadlocks[0].InvolvedThreads[0] != "blocked-worker" {
		t.Fatalf("expected deadlock event, got %+v", deadlocks)
	}
}
