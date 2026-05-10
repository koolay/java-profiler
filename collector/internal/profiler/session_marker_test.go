package profiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestSessionMarkerRoundTripsUnderTargetTmp(t *testing.T) {
	procRoot := t.TempDir()
	startedAt := time.Unix(100, 123).UTC()
	marker := SessionMarker{
		CollectorID: "collector-a",
		PID:         42,
		StartedAt:   startedAt,
		LibraryPath: "/tmp/java-profiler/libasyncProfiler.so",
	}

	if err := WriteSessionMarker(procRoot, 42, marker); err != nil {
		t.Fatal(err)
	}

	got, err := ReadSessionMarker(procRoot, 42)
	if err != nil {
		t.Fatal(err)
	}
	if got.CollectorID != marker.CollectorID || got.PID != marker.PID || got.LibraryPath != marker.LibraryPath || !got.StartedAt.Equal(startedAt) {
		t.Fatalf("marker round trip mismatch: got %#v want %#v", got, marker)
	}
	if _, err := os.Stat(filepath.Join(procRoot, "42", "root", "tmp", "java-profiler-session.json")); err != nil {
		t.Fatalf("expected stable marker path: %v", err)
	}
}

func TestRunnerRecoverOwnedProfilerConflictStopsAndRemovesMarker(t *testing.T) {
	procRoot := t.TempDir()
	startedAt := time.Unix(100, 0).UTC()
	if err := WriteSessionMarker(procRoot, 42, SessionMarker{
		CollectorID: "collector-a",
		PID:         42,
		StartedAt:   startedAt,
		LibraryPath: "/tmp/java-profiler/libasyncProfiler.so",
	}); err != nil {
		t.Fatal(err)
	}
	attach := &recordingAttachController{}
	runner := NewRunner(Config{
		ProcRoot:     procRoot,
		OwnerID:      "collector-a",
		TargetTmpDir: "/tmp/java-profiler",
	}, attach)

	result, err := runner.RecoverConflict(context.Background(), domain.TargetIdentity{ProcessID: 42})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Owned || !result.Recovered {
		t.Fatalf("expected owned recovered conflict, got %#v", result)
	}
	if len(attach.commands) != 1 {
		t.Fatalf("expected one stop command, got %#v", attach.commands)
	}
	if attach.commands[0].pid != 42 || attach.commands[0].args != "stop" {
		t.Fatalf("expected stop against target pid, got %#v", attach.commands[0])
	}
	if _, err := ReadSessionMarker(procRoot, 42); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected recovered marker removed, err=%v", err)
	}
}

func TestRunnerRecoverConflictTreatsMissingDifferentOrInvalidMarkerAsExternal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, procRoot string)
	}{
		{
			name: "missing marker",
		},
		{
			name: "different owner",
			setup: func(t *testing.T, procRoot string) {
				t.Helper()
				if err := WriteSessionMarker(procRoot, 42, SessionMarker{CollectorID: "collector-b", PID: 42}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "invalid marker",
			setup: func(t *testing.T, procRoot string) {
				t.Helper()
				path := filepath.Join(procRoot, "42", "root", "tmp", "java-profiler-session.json")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("{invalid-json"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "different pid",
			setup: func(t *testing.T, procRoot string) {
				t.Helper()
				if err := WriteSessionMarker(procRoot, 42, SessionMarker{CollectorID: "collector-a", PID: 99}); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			procRoot := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, procRoot)
			}
			attach := &recordingAttachController{}
			runner := NewRunner(Config{ProcRoot: procRoot, OwnerID: "collector-a", TargetTmpDir: "/tmp/java-profiler"}, attach)

			result, err := runner.RecoverConflict(context.Background(), domain.TargetIdentity{ProcessID: 42})
			if err != nil {
				t.Fatal(err)
			}
			if result.Owned || result.Recovered {
				t.Fatalf("expected external conflict, got %#v", result)
			}
			if len(attach.commands) != 0 {
				t.Fatalf("expected no stop command for external conflict, got %#v", attach.commands)
			}
		})
	}
}

func TestRunnerRemovesSessionMarkerWhenStartFails(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{
		ProcRoot:     procRoot,
		OwnerID:      "collector-a",
		LibraryPath:  libPath,
		TargetTmpDir: "/tmp/java-profiler",
		Now:          fixedTimes(time.Unix(100, 0)),
	}, failingAttachController{})

	_, err := runner.Collect(context.Background(), "batch-1", domain.TargetIdentity{ProcessID: 42})
	if err == nil {
		t.Fatal("expected start failure")
	}
	if _, markerErr := ReadSessionMarker(procRoot, 42); !errors.Is(markerErr, os.ErrNotExist) {
		t.Fatalf("expected marker removed after start failure, err=%v", markerErr)
	}
}

type failingAttachController struct{}

func (failingAttachController) LoadNativeAgent(context.Context, int, string, string) error {
	return fmt.Errorf("attach failed")
}
