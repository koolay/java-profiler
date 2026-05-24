package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
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
	Container   string
	ProcessID   int
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
	SampleCount int
	EndedAt     time.Time
}

type ProfileTargetSummary struct {
	Namespace        string                       `json:"namespace"`
	Service          string                       `json:"service"`
	Pod              string                       `json:"pod"`
	Container        string                       `json:"container"`
	ProcessID        int                          `json:"process_id"`
	JVMStartTime     time.Time                    `json:"jvm_start_time"`
	ProfileType      domain.ProfileType           `json:"profile_type"`
	TotalValue       uint64                       `json:"total_value"`
	DisplayValue     string                       `json:"display_value"`
	SampleCount      int                          `json:"sample_count"`
	NewestProfileEnd time.Time                    `json:"newest_profile_end"`
	PercentOfTotal   string                       `json:"percent_of_total"`
	WindowSemantics  domain.ProfileValueSemantics `json:"semantics"`
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
	type aggregate struct {
		profileType domain.ProfileType
		frames      []string
		value       uint64
		count       int
		endedAt     time.Time
	}
	byStack := map[string]*aggregate{}
	for _, sample := range r.samples {
		if !profileSampleMatches(sample, q) {
			continue
		}
		key := sample.ProfileType.String() + "\x00" + strings.Join(sample.Frames, "\x00")
		current := byStack[key]
		if current == nil {
			current = &aggregate{profileType: sample.ProfileType, frames: sample.Frames}
			byStack[key] = current
		}
		current.value += sample.Value
		current.count++
		if sample.EndedAt.After(current.endedAt) {
			current.endedAt = sample.EndedAt
		}
	}
	out := make([]TopStackSample, 0, len(byStack))
	for _, item := range byStack {
		out = append(out, TopStackSample{
			ProfileType: item.profileType,
			Frames:      item.frames,
			Value:       item.value,
			SampleCount: item.count,
			EndedAt:     item.endedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return strings.Join(out[i].Frames, "\x00") < strings.Join(out[j].Frames, "\x00")
	})
	if len(out) > limit {
		out = out[:limit]
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
		newest time.Time
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
		if sample.EndedAt.After(current.newest) {
			current.newest = sample.EndedAt
		}
		byTarget[key] = current
		grandTotal += sample.Value
	}
	out := make([]ProfileTargetSummary, 0, len(byTarget))
	window := domain.TimeWindow{StartedAt: q.Start, EndsAt: q.End}
	for _, item := range byTarget {
		out = append(out, ProfileTargetSummary{
			Namespace:        item.sample.Target.Namespace,
			Service:          item.sample.Target.Service,
			Pod:              item.sample.Target.Pod,
			Container:        item.sample.Target.Container,
			ProcessID:        item.sample.Target.ProcessID,
			JVMStartTime:     item.sample.Target.JVMStartTime,
			ProfileType:      item.sample.ProfileType,
			TotalValue:       item.total,
			DisplayValue:     domain.FormatProfileValue(item.sample.ProfileType, item.total, window),
			SampleCount:      item.count,
			NewestProfileEnd: item.newest,
			PercentOfTotal:   percentOfTotal(item.total, grandTotal),
			WindowSemantics:  item.sample.ProfileType.Semantics(window),
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
	if q.Container != "" && sample.Target.Container != q.Container {
		return false
	}
	if q.ProcessID > 0 && sample.Target.ProcessID != q.ProcessID {
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
