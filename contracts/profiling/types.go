package profiling

import (
	"time"

	"github.com/koolay/java-profiler/domain"
)

type ProfileSample struct {
	BatchID     string                `json:"batch_id"`
	Target      domain.TargetIdentity `json:"target"`
	ProfileType domain.ProfileType    `json:"profile_type"`
	StartedAt   time.Time             `json:"started_at"`
	EndedAt     time.Time             `json:"ended_at"`
	StackID     string                `json:"stack_id"`
	Frames      []string              `json:"frames"`
	Value       uint64                `json:"value"`
	Truncated   bool                  `json:"truncated"`
}

type ProfileBatchMetadata struct {
	WindowRawSampleCount        int  `json:"window_raw_sample_count"`
	WindowAggregatedSampleCount int  `json:"window_aggregated_sample_count"`
	BatchSampleCount            int  `json:"batch_sample_count"`
	DroppedSampleCount          int  `json:"dropped_sample_count"`
	DroppedStackCount           int  `json:"dropped_stack_count"`
	Truncated                   bool `json:"truncated"`
	PartIndex                   int  `json:"part_index"`
	PartCount                   int  `json:"part_count"`
}

type ThreadSnapshot struct {
	BatchID         string                `json:"batch_id"`
	Target          domain.TargetIdentity `json:"target"`
	SnapshotAt      time.Time             `json:"snapshot_at"`
	ThreadID        int64                 `json:"thread_id"`
	NativeThreadID  string                `json:"native_thread_id"`
	ThreadName      string                `json:"thread_name"`
	Daemon          bool                  `json:"daemon"`
	State           string                `json:"state"`
	StackFrames     []string              `json:"stack_frames"`
	LockOwner       string                `json:"lock_owner"`
	BlockedLock     string                `json:"blocked_lock"`
	WaitedLock      string                `json:"waited_lock"`
	DeadlockCycleID string                `json:"deadlock_cycle_id"`
	CPUTimeNS       *uint64               `json:"cpu_time_ns"`
	UserCPUTimeNS   *uint64               `json:"user_cpu_time_ns"`
}

type DeadlockEvent struct {
	EventID         string                `json:"event_id"`
	Target          domain.TargetIdentity `json:"target"`
	EventAt         time.Time             `json:"event_at"`
	CycleID         string                `json:"cycle_id"`
	InvolvedThreads []string              `json:"involved_threads"`
	Locks           []string              `json:"locks"`
	BlockingFrames  []string              `json:"blocking_frames"`
}

type JVMEvent struct {
	EventID     string                `json:"event_id"`
	BatchID     string                `json:"batch_id"`
	Target      domain.TargetIdentity `json:"target"`
	EventType   string                `json:"event_type"`
	EventAt     time.Time             `json:"event_at"`
	DurationNS  uint64                `json:"duration_ns"`
	Collector   string                `json:"collector,omitempty"`
	Action      string                `json:"action,omitempty"`
	Cause       string                `json:"cause,omitempty"`
	Message     string                `json:"message,omitempty"`
	StackFrames []string              `json:"stack_frames,omitempty"`
}
