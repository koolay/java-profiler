package domain

import (
	"testing"
	"time"
)

func TestProfileValueSemanticsAndFormatting(t *testing.T) {
	window := TimeWindow{StartedAt: time.Unix(0, 0), EndsAt: time.Unix(10, 0)}

	semantics := ProfileTypeCPU.Semantics(window)
	if semantics.ValueUnit != "nanoseconds" || semantics.DisplayUnit != "duration_and_average_cores" {
		t.Fatalf("unexpected CPU semantics: %+v", semantics)
	}
	if semantics.WindowSeconds != 10 {
		t.Fatalf("window seconds = %v, want 10", semantics.WindowSeconds)
	}
	if got := FormatProfileValue(ProfileTypeCPU, uint64(2*time.Second), window); got != "2.00 s · 0.20 cores" {
		t.Fatalf("CPU display = %q", got)
	}
	if got := FormatProfileValue(ProfileTypeAllocBytes, 2*1024*1024, window); got != "2.0 MiB" {
		t.Fatalf("allocation display = %q", got)
	}
	if got := FormatProfileValue(ProfileTypeLockDelay, uint64(15*time.Millisecond), window); got != "15.0 ms" {
		t.Fatalf("lock delay display = %q", got)
	}
	if got := FormatProfileValue(ProfileTypeWallClock, uint64(3*time.Second), window); got != "3.00 s" {
		t.Fatalf("wall display = %q", got)
	}
	if got := FormatProfileValue(ProfileTypeIOWait, uint64(250*time.Millisecond), window); got != "250.0 ms" {
		t.Fatalf("io wait display = %q", got)
	}
}
