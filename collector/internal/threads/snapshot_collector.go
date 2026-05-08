package threads

import (
	"context"
	"time"
)

type RawSnapshot struct {
	CapturedAtMillis            int64       `json:"capturedAtMillis"`
	ThreadCPUTimeSupported      bool        `json:"threadCpuTimeSupported"`
	ContentionMonitoringEnabled bool        `json:"contentionMonitoringEnabled"`
	Threads                     []RawThread `json:"threads"`
	DeadlockedThreadIDs         []int64     `json:"deadlockedThreadIds"`
}

type RawThread struct {
	ID            int64    `json:"id"`
	NativeID      string   `json:"nativeId"`
	Name          string   `json:"name"`
	Daemon        bool     `json:"daemon"`
	State         string   `json:"state"`
	LockOwner     string   `json:"lockOwner"`
	LockName      string   `json:"lockName"`
	CPUTimeNanos  int64    `json:"cpuTimeNanos"`
	UserTimeNanos int64    `json:"userTimeNanos"`
	Stack         []string `json:"stack"`
}

type SnapshotRunner interface {
	Capture(ctx context.Context, pid int, maxDepth int) (RawSnapshot, error)
}

type Schedule struct {
	Interval  time.Duration
	ExpiresAt time.Time
}

func (s Schedule) Active(now time.Time) bool {
	if s.Interval <= 0 {
		return false
	}
	return s.ExpiresAt.IsZero() || now.Before(s.ExpiresAt)
}
