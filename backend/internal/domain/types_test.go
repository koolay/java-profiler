package domain

import (
	"testing"
	"time"
)

func TestAllProfileTypesAreStableAndValid(t *testing.T) {
	if len(AllProfileTypes) != 5 {
		t.Fatalf("expected 5 profile types, got %d", len(AllProfileTypes))
	}
	for _, pt := range AllProfileTypes {
		if !pt.IsValid() {
			t.Fatalf("profile type %q should be valid", pt)
		}
	}
	want := []string{
		"java_allocation_bytes",
		"java_allocation_objects",
		"java_cpu_nanoseconds",
		"java_lock_contention_count",
		"java_lock_delay_nanoseconds",
	}
	got := StableProfileTypeNames()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stable profile type names changed: got %v want %v", got, want)
		}
	}
}

func TestTargetIdentityKeyUsesJvmStartTime(t *testing.T) {
	base := TargetIdentity{
		Cluster:      "c1",
		Namespace:    "ns",
		Workload:     "wl",
		Pod:          "pod",
		Container:    "ctr",
		Node:         "node",
		PodUID:       "uid1",
		ProcessID:    123,
		JVMStartTime: time.Unix(100, 0).UTC(),
		Service:      "svc",
	}
	other := base
	other.JVMStartTime = time.Unix(200, 0).UTC()
	if base.Key() == other.Key() {
		t.Fatalf("expected different keys when JVM start time changes")
	}
}

func TestDefaultRetentionPolicyBounded(t *testing.T) {
	policy := DefaultRetentionPolicy()
	if !policy.IsBoundedToSevenDays() {
		t.Fatalf("default retention should stay bounded to seven days")
	}
	if got := policy.ArtifactIndexData; got != 24*time.Hour {
		t.Fatalf("expected artifact retention to be 24h, got %s", got)
	}
}
