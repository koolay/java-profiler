package jfr

import (
	"sort"
	"strings"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func AggregateSamples(samples []profiling.ProfileSample) []profiling.ProfileSample {
	if len(samples) == 0 {
		return nil
	}

	type key struct {
		cluster        string
		namespace      string
		workload       string
		pod            string
		container      string
		node           string
		podUID         string
		processID      int
		jvmStartTime   int64
		runtimeVendor  string
		runtimeVersion string
		service        string
		profileType    domain.ProfileType
		startedAt      int64
		endedAt        int64
		stackID        string
	}

	aggregated := make(map[key]profiling.ProfileSample, len(samples))
	for _, sample := range samples {
		k := key{
			cluster:        sample.Target.Cluster,
			namespace:      sample.Target.Namespace,
			workload:       sample.Target.Workload,
			pod:            sample.Target.Pod,
			container:      sample.Target.Container,
			node:           sample.Target.Node,
			podUID:         sample.Target.PodUID,
			processID:      sample.Target.ProcessID,
			jvmStartTime:   sample.Target.JVMStartTime.UnixNano(),
			runtimeVendor:  sample.Target.RuntimeVendor,
			runtimeVersion: sample.Target.RuntimeVersion,
			service:        sample.Target.Service,
			profileType:    sample.ProfileType,
			startedAt:      sample.StartedAt.UnixNano(),
			endedAt:        sample.EndedAt.UnixNano(),
			stackID:        sample.StackID,
		}
		existing, ok := aggregated[k]
		if !ok {
			aggregated[k] = sample
			continue
		}
		existing.Value += sample.Value
		existing.Truncated = existing.Truncated || sample.Truncated
		aggregated[k] = existing
	}

	out := make([]profiling.ProfileSample, 0, len(aggregated))
	for _, sample := range aggregated {
		out = append(out, sample)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ProfileType != out[j].ProfileType {
			return out[i].ProfileType < out[j].ProfileType
		}
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		if framesI, framesJ := strings.Join(out[i].Frames, "\n"), strings.Join(out[j].Frames, "\n"); framesI != framesJ {
			return framesI < framesJ
		}
		if targetI, targetJ := out[i].Target.Key(), out[j].Target.Key(); targetI != targetJ {
			return targetI < targetJ
		}
		if startedI, startedJ := out[i].StartedAt.UnixNano(), out[j].StartedAt.UnixNano(); startedI != startedJ {
			return startedI < startedJ
		}
		if endedI, endedJ := out[i].EndedAt.UnixNano(), out[j].EndedAt.UnixNano(); endedI != endedJ {
			return endedI < endedJ
		}
		return out[i].StackID < out[j].StackID
	})
	return out
}
