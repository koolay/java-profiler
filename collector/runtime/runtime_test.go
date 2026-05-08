package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestRuntimeScanOnceUpdatesStatusesAndMetrics(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "123"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  0 0 0 0 0 0 0 0 0 0\nbtime 1000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "123", "cmdline"), []byte("java\x00-jar\x00app.jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "123", "stat"), []byte("123 (java) S 1 1 1 0 -1 4194560 0 0 0 0 0 0 0 0 20 0 1 0 200 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "123", "maps"), []byte("7f000000-7f100000 r-xp 00000000 00:00 0 /usr/lib/jvm/libjvm.so hotspot"), 0o644); err != nil {
		t.Fatal(err)
	}

	rt := NewCollector(Config{
		ProcRoot:    root,
		CollectorID: "collector-1",
	})
	if err := rt.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	statuses := rt.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	status := statuses[0]
	if status.State != domain.TargetDesiredStateDisabled {
		t.Fatalf("expected disabled state, got %s", status.State)
	}
	if status.Reason != domain.StatusReasonDisabledByMetadata {
		t.Fatalf("expected disabled_by_metadata reason, got %s", status.Reason)
	}
	if status.Target.ProcessID != 123 {
		t.Fatalf("expected pid 123, got %d", status.Target.ProcessID)
	}
	if status.Target.JVMStartTime != time.Unix(1002, 0).UTC() {
		t.Fatalf("expected parsed JVM start time, got %s", status.Target.JVMStartTime)
	}

	snapshot := rt.Exporter().Snapshot()
	for _, want := range []string{
		"java_profiler_collector_up 1",
		"java_profiler_collector_discovered_processes 1",
		"java_profiler_collector_compatible_processes 0",
		"java_profiler_collector_status_entries 1",
		"java_profiler_collector_target_status_disabled_by_metadata 1",
	} {
		if !strings.Contains(snapshot, want) {
			t.Fatalf("expected metrics snapshot to contain %q, got %q", want, snapshot)
		}
	}
}
