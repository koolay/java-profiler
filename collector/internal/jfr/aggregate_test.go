package jfr

import (
	"testing"
	"time"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func TestAggregateSamplesSumsDuplicateStackValues(t *testing.T) {
	now := time.Unix(1, 0)
	target := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo", Service: "jdk17-http-demo"}
	samples := []profiling.ProfileSample{
		{Target: target, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
		{Target: target, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 5, Truncated: true},
	}

	got := AggregateSamples(samples)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Value != 8 {
		t.Fatalf("value = %d", got[0].Value)
	}
	if !got[0].Truncated {
		t.Fatalf("truncated = false")
	}
	if len(got[0].Frames) != 2 || got[0].Frames[1] != "Demo.burnCpu:188" {
		t.Fatalf("frames = %+v", got[0].Frames)
	}
}

func TestAggregateSamplesKeepsDifferentProfileTypesSeparate(t *testing.T) {
	now := time.Unix(1, 0)
	target := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo", Service: "jdk17-http-demo"}
	samples := []profiling.ProfileSample{
		{Target: target, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
		{Target: target, ProfileType: domain.ProfileTypeAllocBytes, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 5},
	}

	got := AggregateSamples(samples)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ProfileType == got[1].ProfileType {
		t.Fatalf("profile types were aggregated together: %+v", got)
	}
}

func TestAggregateSamplesKeepsDifferentTargetsSeparate(t *testing.T) {
	now := time.Unix(1, 0)
	targetA := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo-a", Service: "jdk17-http-demo"}
	targetB := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo-b", Service: "jdk17-http-demo"}
	samples := []profiling.ProfileSample{
		{Target: targetA, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
		{Target: targetB, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 5},
	}

	got := AggregateSamples(samples)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Target.Pod == got[1].Target.Pod {
		t.Fatalf("targets were aggregated together: %+v", got)
	}
}

func TestAggregateSamplesSortsEqualFrameTiesByTarget(t *testing.T) {
	now := time.Unix(1, 0)
	targetA := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo-a", Service: "jdk17-http-demo"}
	targetB := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo-b", Service: "jdk17-http-demo"}
	samples := []profiling.ProfileSample{
		{Target: targetB, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "b", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
		{Target: targetA, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
	}

	got := AggregateSamples(samples)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Target.Pod != "demo-a" || got[1].Target.Pod != "demo-b" {
		t.Fatalf("pods = %q, %q", got[0].Target.Pod, got[1].Target.Pod)
	}
}
