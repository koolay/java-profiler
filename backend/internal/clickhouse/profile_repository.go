package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

type ProfileTargetSummary struct {
	Namespace       string                       `json:"namespace"`
	Service         string                       `json:"service"`
	Pod             string                       `json:"pod"`
	Container       string                       `json:"container"`
	ProcessID       int                          `json:"process_id"`
	JVMStartTime    time.Time                    `json:"jvm_start_time"`
	ProfileType     domain.ProfileType           `json:"profile_type"`
	TotalValue      uint64                       `json:"total_value"`
	DisplayValue    string                       `json:"display_value"`
	SampleCount     int                          `json:"sample_count"`
	PercentOfTotal  string                       `json:"percent_of_total"`
	WindowSemantics domain.ProfileValueSemantics `json:"semantics"`
}

type ProfileSelector struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Pod       string `json:"pod"`
}

type JVMEventQuery struct {
	Namespace string
	Service   string
	Pod       string
	EventType string
	Start     time.Time
	End       time.Time
	Limit     int
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

func (r *ProfileRepository) QueryProfileTargetSummary(_ context.Context, q ProfileQuery) ([]ProfileTargetSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type aggregate struct {
		sample ProfileSample
		total  uint64
		count  int
	}
	byTarget := map[string]aggregate{}
	var grandTotal uint64
	for _, sample := range r.samples {
		if !profileSampleMatches(sample, q) {
			continue
		}
		key := sample.Target.Key() + "|" + sample.ProfileType.String()
		current := byTarget[key]
		current.sample = sample
		current.total += sample.Value
		current.count++
		byTarget[key] = current
		grandTotal += sample.Value
	}
	out := make([]ProfileTargetSummary, 0, len(byTarget))
	window := domain.TimeWindow{StartedAt: q.Start, EndsAt: q.End}
	for _, item := range byTarget {
		out = append(out, ProfileTargetSummary{
			Namespace:       item.sample.Target.Namespace,
			Service:         item.sample.Target.Service,
			Pod:             item.sample.Target.Pod,
			Container:       item.sample.Target.Container,
			ProcessID:       item.sample.Target.ProcessID,
			JVMStartTime:    item.sample.Target.JVMStartTime,
			ProfileType:     item.sample.ProfileType,
			TotalValue:      item.total,
			DisplayValue:    domain.FormatProfileValue(item.sample.ProfileType, item.total, window),
			SampleCount:     item.count,
			PercentOfTotal:  percentOfTotal(item.total, grandTotal),
			WindowSemantics: item.sample.ProfileType.Semantics(window),
		})
	}
	return out, nil
}

func (r *ProfileRepository) QueryProfileSelectors(_ context.Context, q ProfileQuery) ([]ProfileSelector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := map[string]ProfileSelector{}
	for _, sample := range r.samples {
		if !profileSampleMatches(sample, q) {
			continue
		}
		key := sample.Target.Namespace + "\x00" + sample.Target.Service + "\x00" + sample.Target.Pod
		seen[key] = ProfileSelector{Namespace: sample.Target.Namespace, Service: sample.Target.Service, Pod: sample.Target.Pod}
	}
	out := make([]ProfileSelector, 0, len(seen))
	for _, selector := range seen {
		out = append(out, selector)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		if out[i].Service != out[j].Service {
			return out[i].Service < out[j].Service
		}
		return out[i].Pod < out[j].Pod
	})
	return out, nil
}

func profileSampleMatches(sample ProfileSample, q ProfileQuery) bool {
	if q.Namespace != "" && sample.Target.Namespace != q.Namespace {
		return false
	}
	if q.Service != "" && sample.Target.Service != q.Service {
		return false
	}
	if q.Pod != "" && sample.Target.Pod != q.Pod {
		return false
	}
	if q.ProfileType != "" && sample.ProfileType != q.ProfileType {
		return false
	}
	if !q.Start.IsZero() && sample.EndedAt.Before(q.Start) {
		return false
	}
	if !q.End.IsZero() && sample.StartedAt.After(q.End) {
		return false
	}
	return true
}

func percentOfTotal(value, total uint64) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(value)/float64(total)*100)
}

func jvmEventMatches(event JVMEvent, q JVMEventQuery) bool {
	if q.Namespace != "" && event.Target.Namespace != q.Namespace {
		return false
	}
	if q.Service != "" && event.Target.Service != q.Service {
		return false
	}
	if q.Pod != "" && event.Target.Pod != q.Pod {
		return false
	}
	if q.EventType != "" && event.EventType != q.EventType {
		return false
	}
	if !q.Start.IsZero() && event.EventAt.Before(q.Start) {
		return false
	}
	if !q.End.IsZero() && event.EventAt.After(q.End) {
		return false
	}
	return true
}
