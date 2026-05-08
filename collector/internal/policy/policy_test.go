package policy

import (
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestEvaluateDefaultsToDisabled(t *testing.T) {
	got := Evaluate(Metadata{ObservedAt: time.Unix(100, 0)})
	if got.DesiredState != domain.TargetDesiredStateDisabled {
		t.Fatalf("expected disabled by default, got %s", got.DesiredState)
	}
	if got.Reason != domain.StatusReasonDisabledByMetadata {
		t.Fatalf("expected disabled_by_metadata, got %s", got.Reason)
	}
}

func TestEvaluateExplicitDisableWins(t *testing.T) {
	got := Evaluate(Metadata{
		Annotations: map[string]string{
			AnnotationProfileMode:     "continuous",
			AnnotationProfileDisabled: "true",
		},
		ObservedAt: time.Unix(100, 0),
	})
	if !got.Disabled {
		t.Fatalf("explicit disable should win")
	}
	if got.DesiredState != domain.TargetDesiredStateDisabled {
		t.Fatalf("expected disabled, got %s", got.DesiredState)
	}
}

func TestEvaluateTemporaryWinsOverContinuous(t *testing.T) {
	got := Evaluate(Metadata{
		Annotations: map[string]string{
			AnnotationProfileMode:     "temporary",
			AnnotationProfileDuration: "10m",
		},
		Labels: map[string]string{
			AnnotationProfileMode: "continuous",
		},
		ObservedAt: time.Unix(100, 0),
	})
	if got.Mode != domain.EnablementTemporary {
		t.Fatalf("expected temporary mode to win, got %s", got.Mode)
	}
	if got.DesiredState != domain.TargetDesiredStateTemporary {
		t.Fatalf("expected temporary desired state, got %s", got.DesiredState)
	}
}

func TestEvaluateInvalidDurationFailsClosed(t *testing.T) {
	got := Evaluate(Metadata{
		Annotations: map[string]string{
			AnnotationProfileMode:     "temporary",
			AnnotationProfileDuration: "not-a-duration",
		},
		ObservedAt: time.Unix(100, 0),
	})
	if got.Reason != domain.StatusReasonInvalidDuration {
		t.Fatalf("expected invalid duration, got %s", got.Reason)
	}
	if !got.ValidationFailure {
		t.Fatalf("invalid duration should be a validation failure")
	}
}

func TestEvaluateTemporaryRequiresDuration(t *testing.T) {
	got := Evaluate(Metadata{
		Annotations: map[string]string{
			AnnotationProfileMode: "temporary",
		},
		StartedAt:  time.Unix(0, 0),
		ObservedAt: time.Unix(10, 0),
	})
	if got.Reason != domain.StatusReasonInvalidDuration {
		t.Fatalf("expected missing temporary duration to be invalid, got %s", got.Reason)
	}
	if got.DesiredState != domain.TargetDesiredStateDisabled || !got.ValidationFailure {
		t.Fatalf("temporary without duration should fail closed: %+v", got)
	}
}

func TestEvaluateExpiredTemporaryWindowDisabled(t *testing.T) {
	got := Evaluate(Metadata{
		Annotations: map[string]string{
			AnnotationProfileMode:     "temporary",
			AnnotationProfileDuration: "1m",
		},
		StartedAt:  time.Unix(0, 0),
		ObservedAt: time.Unix(120, 0),
	})
	if got.Reason != domain.StatusReasonTemporaryExpired {
		t.Fatalf("expected expired temporary profile, got %s", got.Reason)
	}
	if got.DesiredState != domain.TargetDesiredStateDisabled {
		t.Fatalf("expired temporary profile should be disabled, got %s", got.DesiredState)
	}
}

func TestStableAnnotationKeysAreStable(t *testing.T) {
	keys := StableAnnotationKeys()
	if len(keys) != 5 {
		t.Fatalf("expected five annotation keys, got %d", len(keys))
	}
}
