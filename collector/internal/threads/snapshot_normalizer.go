package threads

import (
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func NormalizeSnapshot(batchID string, target domain.TargetIdentity, raw RawSnapshot) ([]profiling.ThreadSnapshot, []profiling.DeadlockEvent) {
	deadlocked := map[int64]bool{}
	for _, id := range raw.DeadlockedThreadIDs {
		deadlocked[id] = true
	}
	capturedAt := time.UnixMilli(raw.CapturedAtMillis).UTC()
	var snapshots []profiling.ThreadSnapshot
	var involved []string
	for _, thread := range raw.Threads {
		var cpu *uint64
		if raw.ThreadCPUTimeSupported && thread.CPUTimeNanos >= 0 {
			v := uint64(thread.CPUTimeNanos)
			cpu = &v
		}
		cycleID := ""
		if deadlocked[thread.ID] {
			cycleID = batchID + "-deadlock"
			involved = append(involved, thread.Name)
		}
		snapshots = append(snapshots, profiling.ThreadSnapshot{
			BatchID:         batchID,
			Target:          target,
			SnapshotAt:      capturedAt,
			ThreadID:        thread.ID,
			NativeThreadID:  thread.NativeID,
			ThreadName:      thread.Name,
			Daemon:          thread.Daemon,
			State:           thread.State,
			StackFrames:     thread.Stack,
			LockOwner:       thread.LockOwner,
			BlockedLock:     thread.LockName,
			DeadlockCycleID: cycleID,
			CPUTimeNS:       cpu,
		})
	}
	var deadlocks []profiling.DeadlockEvent
	if len(involved) > 0 {
		deadlocks = append(deadlocks, profiling.DeadlockEvent{
			EventID:         batchID + "-deadlock",
			Target:          target,
			EventAt:         capturedAt,
			CycleID:         batchID + "-deadlock",
			InvolvedThreads: involved,
		})
	}
	return snapshots, deadlocks
}
