package profiling

import (
	"time"

	"github.com/koolay/java-profiler/domain"
)

type ProfileSample struct {
	BatchID     string
	Target      domain.TargetIdentity
	ProfileType domain.ProfileType
	StartedAt   time.Time
	EndedAt     time.Time
	StackID     string
	Frames      []string
	Value       uint64
	Truncated   bool
}

type ThreadSnapshot struct {
	BatchID         string
	Target          domain.TargetIdentity
	SnapshotAt      time.Time
	ThreadID        int64
	NativeThreadID  string
	ThreadName      string
	Daemon          bool
	State           string
	StackFrames     []string
	LockOwner       string
	BlockedLock     string
	WaitedLock      string
	DeadlockCycleID string
	CPUTimeNS       *uint64
	UserCPUTimeNS   *uint64
}

type DeadlockEvent struct {
	EventID         string
	Target          domain.TargetIdentity
	EventAt         time.Time
	CycleID         string
	InvolvedThreads []string
	Locks           []string
	BlockingFrames  []string
}
