package status

import (
	"sync"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type TargetStatus struct {
	Target   domain.TargetIdentity
	StatusAt time.Time
	State    domain.TargetDesiredState
	Reason   domain.StatusReason
	Message  string
}

type Store struct {
	mu       sync.RWMutex
	statuses map[string]TargetStatus
}

func NewStore() *Store {
	return &Store{statuses: map[string]TargetStatus{}}
}

func (s *Store) Set(status TargetStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[status.Target.Key()] = status
}

func (s *Store) Snapshot() []TargetStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TargetStatus, 0, len(s.statuses))
	for _, status := range s.statuses {
		out = append(out, status)
	}
	return out
}
