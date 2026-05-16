package clickhouse

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type TargetStatus struct {
	BatchID      string                    `json:"batch_id"`
	Target       domain.TargetIdentity     `json:"target"`
	StatusAt     time.Time                 `json:"status_at"`
	DesiredState domain.TargetDesiredState `json:"desired_state"`
	Reason       domain.StatusReason       `json:"reason"`
	Message      string                    `json:"message"`
}

type TargetStatusQuery struct {
	Namespace string
	Service   string
	Start     time.Time
	End       time.Time
	Limit     int
}

func (s TargetStatus) DesiredStateIsValid() bool {
	switch s.DesiredState {
	case domain.TargetDesiredStateDisabled, domain.TargetDesiredStateEnabled, domain.TargetDesiredStateTemporary, domain.TargetDesiredStateUnsupported:
		return true
	default:
		return false
	}
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

func (r *StatusRepository) LatestByService(_ context.Context, query TargetStatusQuery) ([]TargetStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	latest := map[string]TargetStatus{}
	for _, status := range r.statuses {
		if query.Namespace != "" && status.Target.Namespace != query.Namespace {
			continue
		}
		if query.Service != "" && status.Target.Service != query.Service {
			continue
		}
		if !query.Start.IsZero() && status.StatusAt.Before(query.Start) {
			continue
		}
		if !query.End.IsZero() && status.StatusAt.After(query.End) {
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
	sort.Slice(out, func(i, j int) bool {
		return out[i].StatusAt.After(out[j].StatusAt)
	})
	if query.Limit > 0 && len(out) > query.Limit {
		out = out[:query.Limit]
	}
	return out, nil
}
