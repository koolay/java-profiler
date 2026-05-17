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
	ProfileTypeWallClock      ProfileType = "java_wall_clock_nanoseconds"
	ProfileTypeIOWait         ProfileType = "java_io_wait_nanoseconds"
)

type ProfileValueSemantics struct {
	ValueUnit           string  `json:"value_unit"`
	DisplayUnit         string  `json:"display_unit"`
	PercentBasis        string  `json:"percent_basis"`
	BaselineDescription string  `json:"baseline_description"`
	WindowSeconds       float64 `json:"window_seconds,omitempty"`
}

var AllProfileTypes = []ProfileType{
	ProfileTypeCPU,
	ProfileTypeAllocBytes,
	ProfileTypeAllocObjects,
	ProfileTypeLockContention,
	ProfileTypeLockDelay,
	ProfileTypeWallClock,
	ProfileTypeIOWait,
}

func (p ProfileType) String() string { return string(p) }

func (p ProfileType) IsValid() bool {
	switch p {
	case ProfileTypeCPU, ProfileTypeAllocBytes, ProfileTypeAllocObjects, ProfileTypeLockContention, ProfileTypeLockDelay, ProfileTypeWallClock, ProfileTypeIOWait:
		return true
	default:
		return false
	}
}

func (p ProfileType) Semantics(window TimeWindow) ProfileValueSemantics {
	semantics := ProfileValueSemantics{
		PercentBasis:        "returned_profile_value",
		BaselineDescription: "Percentage is relative to the returned profile samples for the selected filters.",
	}
	if duration := window.Duration(); duration > 0 {
		semantics.WindowSeconds = duration.Seconds()
		semantics.BaselineDescription = "Percentage is relative to returned profile samples; average cores use the selected time window."
	}
	switch p {
	case ProfileTypeCPU:
		semantics.ValueUnit = "nanoseconds"
		semantics.DisplayUnit = "duration_and_average_cores"
	case ProfileTypeWallClock:
		semantics.ValueUnit = "nanoseconds"
		semantics.DisplayUnit = "duration"
		semantics.BaselineDescription = "Percentage is relative to returned Wall Clock samples for the selected Java target and time range."
	case ProfileTypeIOWait:
		semantics.ValueUnit = "nanoseconds"
		semantics.DisplayUnit = "duration"
		semantics.BaselineDescription = "Percentage is relative to returned Java I/O wait samples for the selected target and time range."
	case ProfileTypeAllocBytes:
		semantics.ValueUnit = "bytes"
		semantics.DisplayUnit = "bytes"
	case ProfileTypeAllocObjects:
		semantics.ValueUnit = "objects"
		semantics.DisplayUnit = "count"
	case ProfileTypeLockContention:
		semantics.ValueUnit = "events"
		semantics.DisplayUnit = "count"
	case ProfileTypeLockDelay:
		semantics.ValueUnit = "nanoseconds"
		semantics.DisplayUnit = "duration"
	default:
		semantics.ValueUnit = "raw"
		semantics.DisplayUnit = "raw"
	}
	return semantics
}

func FormatProfileValue(profileType ProfileType, value uint64, window TimeWindow) string {
	switch profileType {
	case ProfileTypeCPU:
		duration := formatDurationNanos(value)
		if windowDuration := window.Duration(); windowDuration > 0 && value > 0 {
			cores := float64(value) / float64(windowDuration.Nanoseconds())
			if cores >= 0.01 {
				return fmt.Sprintf("%s · %.2f cores", duration, cores)
			}
		}
		return duration
	case ProfileTypeLockDelay, ProfileTypeWallClock, ProfileTypeIOWait:
		return formatDurationNanos(value)
	case ProfileTypeAllocBytes:
		return formatBytes(value)
	case ProfileTypeAllocObjects, ProfileTypeLockContention:
		return fmt.Sprintf("%d", value)
	default:
		return fmt.Sprintf("%d", value)
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
	StatusReasonNone                    StatusReason = ""
	StatusReasonDisabledByMetadata      StatusReason = "disabled_by_metadata"
	StatusReasonTemporaryExpired        StatusReason = "temporary_expired"
	StatusReasonInvalidDuration         StatusReason = "invalid_duration"
	StatusReasonUnsupportedJVM          StatusReason = "unsupported_jvm"
	StatusReasonProfilerConflict        StatusReason = "profiler_conflict"
	StatusReasonOrphanedProfilerSession StatusReason = "orphaned_profiler_session"
	StatusReasonAttachFailed            StatusReason = "attach_failed"
	StatusReasonUploadRetryable         StatusReason = "upload_retryable"
	StatusReasonUploadDropped           StatusReason = "upload_dropped"
	StatusReasonStorageRejected         StatusReason = "storage_rejected"
	StatusReasonAccepted                StatusReason = "accepted"
)

func (r StatusReason) IsValid() bool {
	switch r {
	case StatusReasonNone, StatusReasonDisabledByMetadata, StatusReasonTemporaryExpired, StatusReasonInvalidDuration,
		StatusReasonUnsupportedJVM, StatusReasonProfilerConflict, StatusReasonOrphanedProfilerSession, StatusReasonAttachFailed, StatusReasonUploadRetryable,
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
	BatchTypeJVMEvent       BatchType = "jvm_event"
	BatchTypeTargetStatus   BatchType = "target_status"
	BatchTypeCollectorBeat  BatchType = "collector_heartbeat"
	BatchTypeIngestion      BatchType = "ingestion"
	BatchTypeRetention      BatchType = "retention"
	BatchTypeArtifactIndex  BatchType = "artifact_index"
)

func (b BatchType) IsValid() bool {
	switch b {
	case BatchTypeProfile, BatchTypeThreadSnapshot, BatchTypeJVMEvent, BatchTypeTargetStatus, BatchTypeCollectorBeat, BatchTypeIngestion, BatchTypeRetention, BatchTypeArtifactIndex:
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

func formatDurationNanos(value uint64) string {
	switch {
	case value >= uint64(time.Minute):
		return fmt.Sprintf("%.1f min", float64(value)/float64(time.Minute))
	case value >= uint64(time.Second):
		return fmt.Sprintf("%.2f s", float64(value)/float64(time.Second))
	case value >= uint64(time.Millisecond):
		return fmt.Sprintf("%.1f ms", float64(value)/float64(time.Millisecond))
	case value >= uint64(time.Microsecond):
		return fmt.Sprintf("%.1f us", float64(value)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%d ns", value)
	}
}

func formatBytes(value uint64) string {
	const unit = 1024
	switch {
	case value >= unit*unit*unit:
		return fmt.Sprintf("%.1f GiB", float64(value)/(unit*unit*unit))
	case value >= unit*unit:
		return fmt.Sprintf("%.1f MiB", float64(value)/(unit*unit))
	case value >= unit:
		return fmt.Sprintf("%.1f KiB", float64(value)/unit)
	default:
		return fmt.Sprintf("%d B", value)
	}
}
