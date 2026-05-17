package jfr

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

const (
	DefaultCPUExecutionSampleValueNS uint64 = 10_000_000
	DefaultWallClockSampleValueNS    uint64 = 10_000_000
	DefaultIOWaitWallSampleValueNS   uint64 = 10_000_000
)

type NormalizedWindow struct {
	Samples        []profiling.ProfileSample
	JVMEvents      []profiling.JVMEvent
	RawSampleCount int
}

func Normalize(batchID string, target domain.TargetIdentity, events []Event) []profiling.ProfileSample {
	return NormalizeWindow(batchID, target, events, time.Time{}, time.Time{})
}

func NormalizeWindow(batchID string, target domain.TargetIdentity, events []Event, startedAt, endedAt time.Time) []profiling.ProfileSample {
	return NormalizeWindowWithStats(batchID, target, events, startedAt, endedAt).Samples
}

func NormalizeWindowWithStats(batchID string, target domain.TargetIdentity, events []Event, startedAt, endedAt time.Time) NormalizedWindow {
	var samples []profiling.ProfileSample
	var jvmEvents []profiling.JVMEvent
	rawSampleCount := 0
	for _, event := range events {
		if jvmEvent, ok := normalizeJVMEvent(batchID, target, event, startedAt, endedAt); ok {
			jvmEvents = append(jvmEvents, jvmEvent)
			continue
		}
		profileType, ok := profileTypeForEvent(event.Type)
		if !ok {
			continue
		}
		rawSampleCount++
		value := event.Value
		switch profileType {
		case domain.ProfileTypeCPU:
			value = event.Value * DefaultCPUExecutionSampleValueNS
		case domain.ProfileTypeWallClock:
			value = event.Value * DefaultWallClockSampleValueNS
		case domain.ProfileTypeIOWait:
			value = event.Value * DefaultIOWaitWallSampleValueNS
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
			Value:       value,
			Truncated:   len(event.Frames) > len(frames),
		})
	}
	return NormalizedWindow{Samples: AggregateSamples(samples), JVMEvents: jvmEvents, RawSampleCount: rawSampleCount}
}

func normalizeJVMEvent(batchID string, target domain.TargetIdentity, event Event, startedAt, endedAt time.Time) (profiling.JVMEvent, bool) {
	switch event.Type {
	case "gc_pause":
	default:
		return profiling.JVMEvent{}, false
	}
	eventAt := endedAt
	if eventAt.IsZero() {
		eventAt = startedAt
	}
	if eventAt.IsZero() {
		eventAt = time.Now().UTC()
	}
	eventID := event.Labels["event_id"]
	if eventID == "" {
		eventID = stackID(append(event.Frames, fmt.Sprintf("%s:%d:%s", event.Type, event.Value, eventAt.UTC().Format(time.RFC3339Nano))))
	}
	return profiling.JVMEvent{
		EventID:     eventID,
		BatchID:     batchID,
		Target:      target,
		EventType:   event.Type,
		EventAt:     eventAt,
		DurationNS:  event.Value,
		Collector:   event.Labels["collector"],
		Action:      event.Labels["action"],
		Cause:       event.Labels["cause"],
		Message:     event.Labels["message"],
		StackFrames: boundedFrames(event.Frames, 256),
	}, true
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
	case "wall_clock":
		return domain.ProfileTypeWallClock, true
	case "io_wait":
		return domain.ProfileTypeIOWait, true
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
