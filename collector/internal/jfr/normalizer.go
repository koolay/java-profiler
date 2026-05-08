package jfr

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func Normalize(batchID string, target domain.TargetIdentity, events []Event) []profiling.ProfileSample {
	return NormalizeWindow(batchID, target, events, time.Time{}, time.Time{})
}

func NormalizeWindow(batchID string, target domain.TargetIdentity, events []Event, startedAt, endedAt time.Time) []profiling.ProfileSample {
	var samples []profiling.ProfileSample
	for _, event := range events {
		profileType, ok := profileTypeForEvent(event.Type)
		if !ok {
			continue
		}
		frames := boundedFrames(event.Frames, 256)
		samples = append(samples, profiling.ProfileSample{
			BatchID:     batchID,
			Target:      target,
			ProfileType: profileType,
			StartedAt:   startedAt,
			EndedAt:     endedAt,
			StackID:     stackID(frames),
			Frames:      frames,
			Value:       event.Value,
			Truncated:   len(event.Frames) > len(frames),
		})
	}
	return samples
}

func profileTypeForEvent(event string) (domain.ProfileType, bool) {
	switch event {
	case "execution_sample":
		return domain.ProfileTypeCPU, true
	case "alloc_bytes":
		return domain.ProfileTypeAllocBytes, true
	case "alloc_objects":
		return domain.ProfileTypeAllocObjects, true
	case "monitor_enter":
		return domain.ProfileTypeLockContention, true
	case "lock_delay":
		return domain.ProfileTypeLockDelay, true
	default:
		return "", false
	}
}

func boundedFrames(frames []string, limit int) []string {
	if len(frames) <= limit {
		return append([]string(nil), frames...)
	}
	return append([]string(nil), frames[:limit]...)
}

func stackID(frames []string) string {
	sum := sha1.Sum([]byte(strings.Join(frames, "\n")))
	return hex.EncodeToString(sum[:])
}
