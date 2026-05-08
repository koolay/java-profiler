package clickhouse

import (
	"context"
	"sync"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type TargetStatus struct {
	BatchID      string
	Target       domain.TargetIdentity
	StatusAt     time.Time
	DesiredState domain.TargetDesiredState
	Reason       domain.StatusReason
	Message      string
}

type StatusRepository struct {
	mu       sync.RWMutex
	statuses []TargetStatus
}

func NewStatusRepository() *StatusRepository { return &StatusRepository{} }

func (r *StatusRepository) InsertStatuses(_ context.Context, statuses []TargetStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses = append(r.statuses, statuses...)
	return nil
}

func (r *StatusRepository) LatestByService(_ context.Context, namespace, service string) ([]TargetStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[string]TargetStatus{}
	for _, status := range r.statuses {
		if namespace != "" && status.Target.Namespace != namespace {
			continue
		}
		if service != "" && status.Target.Service != service {
			continue
		}
		key := status.Target.Key()
		if prior, ok := latest[key]; !ok || status.StatusAt.After(prior.StatusAt) {
			latest[key] = status
		}
	}
	out := make([]TargetStatus, 0, len(latest))
	for _, status := range latest {
		out = append(out, status)
	}
	return out, nil
}
