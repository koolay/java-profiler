package policy

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/koolay/java-profiler/domain"
)

const (
	AnnotationProfileMode      = "java-profiler.io/profile-mode"
	AnnotationProfileDisabled  = "java-profiler.io/profile-disabled"
	AnnotationProfileDuration  = "java-profiler.io/profile-duration"
	AnnotationStartupDelay     = "java-profiler.io/startup-delay"
	AnnotationSnapshotInterval = "java-profiler.io/snapshot-interval"

	LabelProfileMode      = AnnotationProfileMode
	LabelProfileDisabled  = AnnotationProfileDisabled
	LabelProfileDuration  = AnnotationProfileDuration
	LabelStartupDelay     = AnnotationStartupDelay
	LabelSnapshotInterval = AnnotationSnapshotInterval
)

type Metadata struct {
	Annotations map[string]string
	Labels      map[string]string
	StartedAt   time.Time
	ObservedAt  time.Time
}

type Evaluation struct {
	DesiredState      domain.TargetDesiredState
	Mode              domain.EnablementMode
	Reason            domain.StatusReason
	Message           string
	TemporaryWindow   domain.TimeWindow
	StartupDelay      time.Duration
	SnapshotInterval  time.Duration
	ProfileTypes      []domain.ProfileType
	Disabled          bool
	ValidationFailure bool
}

func Evaluate(meta Metadata) Evaluation {
	values := merge(meta.Annotations, meta.Labels)
	eval := Evaluation{
		DesiredState:     domain.TargetDesiredStateDisabled,
		Mode:             domain.EnablementDisabled,
		Reason:           domain.StatusReasonDisabledByMetadata,
		StartupDelay:     30 * time.Second,
		SnapshotInterval: 5 * time.Minute,
		ProfileTypes:     append([]domain.ProfileType(nil), domain.AllProfileTypes...),
	}

	if isTruthy(values[AnnotationProfileDisabled]) {
		eval.Disabled = true
		eval.Message = "profiling disabled explicitly"
		return eval
	}

	mode := normalizeMode(values[AnnotationProfileMode])
	switch mode {
	case "temporary":
		eval.Mode = domain.EnablementTemporary
		eval.DesiredState = domain.TargetDesiredStateTemporary
		eval.Reason = domain.StatusReasonAccepted
		eval.Message = "temporary profiling enabled"
	case "continuous":
		eval.Mode = domain.EnablementContinuous
		eval.DesiredState = domain.TargetDesiredStateEnabled
		eval.Reason = domain.StatusReasonAccepted
		eval.Message = "continuous profiling enabled"
	default:
		eval.Mode = domain.EnablementDisabled
		eval.DesiredState = domain.TargetDesiredStateDisabled
		eval.Reason = domain.StatusReasonDisabledByMetadata
		eval.Message = "profiling not enabled"
		return eval
	}

	if duration := strings.TrimSpace(values[AnnotationProfileDuration]); duration != "" {
		parsed, err := time.ParseDuration(duration)
		if err != nil {
			eval.DesiredState = domain.TargetDesiredStateDisabled
			eval.Reason = domain.StatusReasonInvalidDuration
			eval.Message = fmt.Sprintf("invalid profile duration %q", duration)
			eval.ValidationFailure = true
			return eval
		}
		if parsed <= 0 {
			eval.DesiredState = domain.TargetDesiredStateDisabled
			eval.Reason = domain.StatusReasonInvalidDuration
			eval.Message = "profile duration must be positive"
			eval.ValidationFailure = true
			return eval
		}
		if !meta.StartedAt.IsZero() {
			eval.TemporaryWindow = domain.TimeWindow{
				StartedAt: meta.StartedAt,
				EndsAt:    meta.StartedAt.Add(parsed),
			}
			if !eval.TemporaryWindow.EndsAt.After(eval.TemporaryWindow.StartedAt) {
				eval.DesiredState = domain.TargetDesiredStateDisabled
				eval.Reason = domain.StatusReasonInvalidDuration
				eval.Message = "profile window must end after it starts"
				eval.ValidationFailure = true
				return eval
			}
		}
	}

	if mode == "temporary" && !eval.TemporaryWindow.IsZero() && !meta.ObservedAt.IsZero() && !meta.ObservedAt.Before(eval.TemporaryWindow.EndsAt) {
		eval.DesiredState = domain.TargetDesiredStateDisabled
		eval.Reason = domain.StatusReasonTemporaryExpired
		eval.Message = "temporary profiling window already expired"
		return eval
	}

	startup := strings.TrimSpace(values[AnnotationStartupDelay])
	if startup != "" {
		parsed, err := time.ParseDuration(startup)
		if err != nil {
			eval.DesiredState = domain.TargetDesiredStateDisabled
			eval.Reason = domain.StatusReasonInvalidDuration
			eval.Message = fmt.Sprintf("invalid startup delay %q", startup)
			eval.ValidationFailure = true
			return eval
		}
		eval.StartupDelay = parsed
	}

	if snapshot := strings.TrimSpace(values[AnnotationSnapshotInterval]); snapshot != "" {
		parsed, err := time.ParseDuration(snapshot)
		if err != nil {
			eval.DesiredState = domain.TargetDesiredStateDisabled
			eval.Reason = domain.StatusReasonInvalidDuration
			eval.Message = fmt.Sprintf("invalid snapshot interval %q", snapshot)
			eval.ValidationFailure = true
			return eval
		}
		eval.SnapshotInterval = parsed
	}

	if mode == "temporary" && !eval.TemporaryWindow.IsZero() && !meta.ObservedAt.IsZero() && meta.ObservedAt.After(eval.TemporaryWindow.EndsAt) {
		eval.DesiredState = domain.TargetDesiredStateDisabled
		eval.Reason = domain.StatusReasonTemporaryExpired
		eval.Message = "temporary profiling window expired"
		return eval
	}

	return eval
}

func merge(annotations, labels map[string]string) map[string]string {
	out := make(map[string]string, len(annotations)+len(labels))
	for k, v := range labels {
		out[k] = v
	}
	for k, v := range annotations {
		out[k] = v
	}
	return out
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "enabled", "on":
		return true
	default:
		return false
	}
}

func normalizeMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "temporary", "temp", "incident":
		return "temporary"
	case "continuous", "enabled", "on", "yes", "true", "1":
		return "continuous"
	default:
		return ""
	}
}

func StableAnnotationKeys() []string {
	keys := []string{
		AnnotationProfileDisabled,
		AnnotationProfileDuration,
		AnnotationProfileMode,
		AnnotationSnapshotInterval,
		AnnotationStartupDelay,
	}
	sort.Strings(keys)
	return keys
}
