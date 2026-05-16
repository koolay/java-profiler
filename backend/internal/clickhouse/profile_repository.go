package clickhouse

import (
	"context"
	"errors"
	"sync"
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

var ErrDuplicateBatch = errors.New("duplicate ingestion batch")
var ErrRetryableStorage = errors.New("retryable storage failure")

type ProfileSample = profiling.ProfileSample

type ProfileQuery struct {
	Namespace   string
	Service     string
	Pod         string
	ProfileType domain.ProfileType
	Start       time.Time
	End         time.Time
	Limit       int
}

type FlamegraphSample struct {
	Frames []string
	Value  uint64
}

type TopStackSample struct {
	ProfileType domain.ProfileType
	Frames      []string
	Value       uint64
}

type ProfileRepository struct {
	mu      sync.RWMutex
	batches map[string]struct{}
	samples []ProfileSample
}

func NewProfileRepository() *ProfileRepository {
	return &ProfileRepository{batches: map[string]struct{}{}}
}

func (r *ProfileRepository) InsertProfileBatch(_ context.Context, batchID string, samples []ProfileSample) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.batches[batchID]; exists {
		return ErrDuplicateBatch
	}
	r.batches[batchID] = struct{}{}
	r.samples = append(r.samples, samples...)
	return nil
}

func (r *ProfileRepository) QuerySamples(_ context.Context, q ProfileQuery) ([]ProfileSample, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	out := make([]ProfileSample, 0)
	for _, sample := range r.samples {
		if q.Namespace != "" && sample.Target.Namespace != q.Namespace {
			continue
		}
		if q.Service != "" && sample.Target.Service != q.Service {
			continue
		}
		if q.Pod != "" && sample.Target.Pod != q.Pod {
			continue
		}
		if q.ProfileType != "" && sample.ProfileType != q.ProfileType {
			continue
		}
		if !q.Start.IsZero() && sample.EndedAt.Before(q.Start) {
			continue
		}
		if !q.End.IsZero() && sample.StartedAt.After(q.End) {
			continue
		}
		out = append(out, sample)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *ProfileRepository) QueryFlamegraphSamples(_ context.Context, q ProfileQuery) ([]FlamegraphSample, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	out := make([]FlamegraphSample, 0)
	for _, sample := range r.samples {
		if q.Namespace != "" && sample.Target.Namespace != q.Namespace {
			continue
		}
		if q.Service != "" && sample.Target.Service != q.Service {
			continue
		}
		if q.Pod != "" && sample.Target.Pod != q.Pod {
			continue
		}
		if q.ProfileType != "" && sample.ProfileType != q.ProfileType {
			continue
		}
		if !q.Start.IsZero() && sample.EndedAt.Before(q.Start) {
			continue
		}
		if !q.End.IsZero() && sample.StartedAt.After(q.End) {
			continue
		}
		out = append(out, FlamegraphSample{Frames: sample.Frames, Value: sample.Value})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *ProfileRepository) QueryTopStackSamples(_ context.Context, q ProfileQuery) ([]TopStackSample, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	out := make([]TopStackSample, 0)
	for _, sample := range r.samples {
		if q.Namespace != "" && sample.Target.Namespace != q.Namespace {
			continue
		}
		if q.Service != "" && sample.Target.Service != q.Service {
			continue
		}
		if q.Pod != "" && sample.Target.Pod != q.Pod {
			continue
		}
		if q.ProfileType != "" && sample.ProfileType != q.ProfileType {
			continue
		}
		if !q.Start.IsZero() && sample.EndedAt.Before(q.Start) {
			continue
		}
		if !q.End.IsZero() && sample.StartedAt.After(q.End) {
			continue
		}
		out = append(out, TopStackSample{ProfileType: sample.ProfileType, Frames: sample.Frames, Value: sample.Value})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
