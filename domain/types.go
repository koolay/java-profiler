package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type ProfileType string

const (
	ProfileTypeCPU            ProfileType = "java_cpu_nanoseconds"
	ProfileTypeAllocBytes     ProfileType = "java_allocation_bytes"
	ProfileTypeAllocObjects   ProfileType = "java_allocation_objects"
	ProfileTypeLockContention ProfileType = "java_lock_contention_count"
	ProfileTypeLockDelay      ProfileType = "java_lock_delay_nanoseconds"
)

var AllProfileTypes = []ProfileType{
	ProfileTypeCPU,
	ProfileTypeAllocBytes,
	ProfileTypeAllocObjects,
	ProfileTypeLockContention,
	ProfileTypeLockDelay,
}

func (p ProfileType) String() string { return string(p) }

func (p ProfileType) IsValid() bool {
	switch p {
	case ProfileTypeCPU, ProfileTypeAllocBytes, ProfileTypeAllocObjects, ProfileTypeLockContention, ProfileTypeLockDelay:
		return true
	default:
		return false
	}
}

type EnablementMode string

const (
	EnablementDisabled   EnablementMode = "disabled"
	EnablementContinuous EnablementMode = "continuous"
	EnablementTemporary  EnablementMode = "temporary"
)

func (m EnablementMode) IsValid() bool {
	switch m {
	case EnablementDisabled, EnablementContinuous, EnablementTemporary:
		return true
	default:
		return false
	}
}

type TargetDesiredState string

const (
	TargetDesiredStateDisabled    TargetDesiredState = "disabled"
	TargetDesiredStateEnabled     TargetDesiredState = "enabled"
	TargetDesiredStateTemporary   TargetDesiredState = "temporary"
	TargetDesiredStateUnsupported TargetDesiredState = "unsupported"
)

type StatusReason string

const (
	StatusReasonNone               StatusReason = ""
	StatusReasonDisabledByMetadata StatusReason = "disabled_by_metadata"
	StatusReasonTemporaryExpired   StatusReason = "temporary_expired"
	StatusReasonInvalidDuration    StatusReason = "invalid_duration"
	StatusReasonUnsupportedJVM     StatusReason = "unsupported_jvm"
	StatusReasonProfilerConflict   StatusReason = "profiler_conflict"
	StatusReasonAttachFailed       StatusReason = "attach_failed"
	StatusReasonUploadRetryable    StatusReason = "upload_retryable"
	StatusReasonUploadDropped      StatusReason = "upload_dropped"
	StatusReasonStorageRejected    StatusReason = "storage_rejected"
	StatusReasonAccepted           StatusReason = "accepted"
)

func (r StatusReason) IsValid() bool {
	switch r {
	case StatusReasonNone, StatusReasonDisabledByMetadata, StatusReasonTemporaryExpired, StatusReasonInvalidDuration,
		StatusReasonUnsupportedJVM, StatusReasonProfilerConflict, StatusReasonAttachFailed, StatusReasonUploadRetryable,
		StatusReasonUploadDropped, StatusReasonStorageRejected, StatusReasonAccepted:
		return true
	default:
		return false
	}
}

type BatchType string

const (
	BatchTypeProfile        BatchType = "profile"
	BatchTypeThreadSnapshot BatchType = "thread_snapshot"
	BatchTypeTargetStatus   BatchType = "target_status"
	BatchTypeCollectorBeat  BatchType = "collector_heartbeat"
	BatchTypeIngestion      BatchType = "ingestion"
	BatchTypeRetention      BatchType = "retention"
	BatchTypeArtifactIndex  BatchType = "artifact_index"
)

func (b BatchType) IsValid() bool {
	switch b {
	case BatchTypeProfile, BatchTypeThreadSnapshot, BatchTypeTargetStatus, BatchTypeCollectorBeat, BatchTypeIngestion, BatchTypeRetention, BatchTypeArtifactIndex:
		return true
	default:
		return false
	}
}

type TargetIdentity struct {
	Cluster        string    `json:"cluster"`
	Namespace      string    `json:"namespace"`
	Workload       string    `json:"workload"`
	Pod            string    `json:"pod"`
	Container      string    `json:"container"`
	Node           string    `json:"node"`
	PodUID         string    `json:"pod_uid"`
	ProcessID      int       `json:"process_id"`
	JVMStartTime   time.Time `json:"jvm_start_time"`
	RuntimeVendor  string    `json:"runtime_vendor"`
	RuntimeVersion string    `json:"runtime_version"`
	Service        string    `json:"service"`
}

func (t TargetIdentity) Key() string {
	return strings.Join([]string{
		t.Cluster,
		t.Namespace,
		t.Workload,
		t.Pod,
		t.Container,
		t.Node,
		t.PodUID,
		fmt.Sprintf("%d", t.ProcessID),
		t.JVMStartTime.UTC().Format(time.RFC3339Nano),
		t.RuntimeVendor,
		t.RuntimeVersion,
		t.Service,
	}, "|")
}

type TimeWindow struct {
	StartedAt time.Time `json:"started_at"`
	EndsAt    time.Time `json:"ends_at"`
}

func (w TimeWindow) IsZero() bool { return w.StartedAt.IsZero() && w.EndsAt.IsZero() }

func (w TimeWindow) Duration() time.Duration {
	if w.EndsAt.Before(w.StartedAt) {
		return 0
	}
	return w.EndsAt.Sub(w.StartedAt)
}

type RetentionPolicy struct {
	ProfileData       time.Duration `json:"profile_data"`
	ThreadData        time.Duration `json:"thread_data"`
	DeadlockData      time.Duration `json:"deadlock_data"`
	TargetStatusData  time.Duration `json:"target_status_data"`
	IngestionData     time.Duration `json:"ingestion_data"`
	ArtifactIndexData time.Duration `json:"artifact_index_data"`
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		ProfileData:       7 * 24 * time.Hour,
		ThreadData:        7 * 24 * time.Hour,
		DeadlockData:      7 * 24 * time.Hour,
		TargetStatusData:  7 * 24 * time.Hour,
		IngestionData:     7 * 24 * time.Hour,
		ArtifactIndexData: 24 * time.Hour,
	}
}

func (r RetentionPolicy) Values() []time.Duration {
	return []time.Duration{
		r.ProfileData,
		r.ThreadData,
		r.DeadlockData,
		r.TargetStatusData,
		r.IngestionData,
		r.ArtifactIndexData,
	}
}

func (r RetentionPolicy) IsBoundedToSevenDays() bool {
	for _, d := range r.Values() {
		if d <= 0 || d > 7*24*time.Hour {
			return false
		}
	}
	return true
}

type Confidence string

const (
	ConfidenceExactThreadCPU  Confidence = "exact_thread_cpu"
	ConfidenceSampledRUNNABLE Confidence = "sampled_runnable"
	ConfidenceProfileOnly     Confidence = "profile_only"
)

func (c Confidence) IsValid() bool {
	switch c {
	case ConfidenceExactThreadCPU, ConfidenceSampledRUNNABLE, ConfidenceProfileOnly:
		return true
	default:
		return false
	}
}

func StableProfileTypeNames() []string {
	out := make([]string, 0, len(AllProfileTypes))
	for _, p := range AllProfileTypes {
		out = append(out, p.String())
	}
	sort.Strings(out)
	return out
}
