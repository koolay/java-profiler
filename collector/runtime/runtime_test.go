package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/koolay/java-profiler/collector/internal/pipeline"
	"github.com/koolay/java-profiler/collector/internal/policy"
	"github.com/koolay/java-profiler/collector/internal/profiler"
	profiling "github.com/koolay/java-profiler/contracts/profiling"
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

func TestChunkProfileSamplesCopiesAndPreservesSmallBatches(t *testing.T) {
	samples := []profiling.ProfileSample{
		{BatchID: "batch", StackID: "a", Value: 1},
		{BatchID: "batch", StackID: "b", Value: 2},
		{BatchID: "batch", StackID: "c", Value: 3},
	}

	small := chunkProfileSamples(samples[:2], 10)
	if len(small) != 1 || len(small[0]) != 2 {
		t.Fatalf("expected one small chunk, got %#v", small)
	}

	chunks := chunkProfileSamples(samples, 2)
	if len(chunks) != 2 || len(chunks[0]) != 2 || len(chunks[1]) != 1 {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
	chunks[0][0].BatchID = "changed"
	if samples[0].BatchID != "batch" {
		t.Fatalf("chunk mutation leaked into source sample: %#v", samples[0])
	}
}

func TestUploadProfileSamplesSendsBoundedMultipartMetadata(t *testing.T) {
	var uploads []pipeline.ProfileBatchPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload pipeline.ProfileBatchPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upload: %v", err)
		}
		uploads = append(uploads, payload)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	rt := NewCollector(Config{
		CollectorID: "collector-a",
		BackendURL:  server.URL,
	})
	rt.profileLimits = pipeline.ProfileBatchLimits{
		MaxSamplesPerWindow: 3,
		MaxSamplesPerBatch:  2,
	}

	err := rt.uploadProfileSamples(context.Background(), "batch-a", []profiling.ProfileSample{
		{BatchID: "batch-a", StackID: "a", Value: 1},
		{BatchID: "batch-a", StackID: "b", Value: 1},
		{BatchID: "batch-a", StackID: "c", Value: 1},
		{BatchID: "batch-a", StackID: "d", Value: 1},
	}, 7)
	if err != nil {
		t.Fatal(err)
	}

	if len(uploads) != 2 {
		t.Fatalf("uploads = %d", len(uploads))
	}

	first := uploads[0]
	if first.BatchID != "batch-a-part-0001" {
		t.Fatalf("first batch id = %q", first.BatchID)
	}
	if len(first.Samples) != 2 {
		t.Fatalf("first sample count = %d", len(first.Samples))
	}
	if first.Samples[0].BatchID != first.BatchID || first.Samples[1].BatchID != first.BatchID {
		t.Fatalf("first sample batch ids = %#v", first.Samples)
	}
	if first.Metadata.WindowRawSampleCount != 7 {
		t.Fatalf("first raw sample count = %d", first.Metadata.WindowRawSampleCount)
	}
	if first.Metadata.WindowAggregatedSampleCount != 4 {
		t.Fatalf("first aggregated sample count = %d", first.Metadata.WindowAggregatedSampleCount)
	}
	if first.Metadata.DroppedSampleCount != 1 || first.Metadata.DroppedStackCount != 1 {
		t.Fatalf("first dropped counts = %#v", first.Metadata)
	}
	if !first.Metadata.Truncated {
		t.Fatalf("first metadata should be truncated")
	}
	if first.Metadata.PartIndex != 1 || first.Metadata.PartCount != 2 || first.Metadata.BatchSampleCount != 2 {
		t.Fatalf("first part metadata = %#v", first.Metadata)
	}

	second := uploads[1]
	if second.BatchID != "batch-a-part-0002" {
		t.Fatalf("second batch id = %q", second.BatchID)
	}
	if len(second.Samples) != 1 {
		t.Fatalf("second sample count = %d", len(second.Samples))
	}
	if second.Samples[0].BatchID != second.BatchID {
		t.Fatalf("second sample batch id = %#v", second.Samples[0])
	}
	if second.Metadata.PartIndex != 2 || second.Metadata.PartCount != 2 || second.Metadata.BatchSampleCount != 1 {
		t.Fatalf("second part metadata = %#v", second.Metadata)
	}
	if second.Metadata.WindowRawSampleCount != 0 ||
		second.Metadata.WindowAggregatedSampleCount != 0 ||
		second.Metadata.DroppedSampleCount != 0 ||
		second.Metadata.DroppedStackCount != 0 ||
		second.Metadata.Truncated {
		t.Fatalf("second part repeated window metadata: %#v", second.Metadata)
	}
}

func TestRuntimeScanOnceRecoversOwnedStaleProfilerConflictAndDefersProfiling(t *testing.T) {
	root := t.TempDir()
	writeRuntimeProcess(t, root, 123, true)
	uid := "11111111-2222-3333-4444-555555555555"
	if err := os.WriteFile(filepath.Join(root, "123", "cgroup"), []byte("0::/kubepods/pod"+uid+"/container"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := profiler.WriteSessionMarker(root, 123, profiler.SessionMarker{
		CollectorID: "collector-1",
		PID:         123,
		StartedAt:   time.Unix(10, 0).UTC(),
		LibraryPath: "/tmp/java-profiler/libasyncProfiler.so",
	}); err != nil {
		t.Fatal(err)
	}
	attach := &recordingRuntimeAttach{}
	rt := NewCollector(Config{ProcRoot: root, CollectorID: "collector-1"})
	rt.profiler = profiler.NewRunner(profiler.Config{
		ProcRoot:     root,
		OwnerID:      "collector-1",
		TargetTmpDir: "/tmp/java-profiler",
	}, attach)
	rt.podSource = func(context.Context) map[string]podItem {
		return map[string]podItem{
			normalizeUID(uid): readyProfiledPod(uid),
		}
	}

	if err := rt.ScanOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	statuses := rt.Statuses()
	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	status := statuses[0]
	if status.Reason != domain.StatusReasonOrphanedProfilerSession {
		t.Fatalf("expected orphaned profiler session reason, got %#v", status)
	}
	if !strings.Contains(status.Message, "recovered") || !strings.Contains(status.Message, "next scan") {
		t.Fatalf("expected recovered/deferred message, got %q", status.Message)
	}
	if len(attach.commands) != 1 || attach.commands[0].args != "stop" || attach.commands[0].pid != 123 {
		t.Fatalf("expected one stop command for owned conflict, got %#v", attach.commands)
	}
	if _, err := profiler.ReadSessionMarker(root, 123); !os.IsNotExist(err) {
		t.Fatalf("expected recovered marker removed, err=%v", err)
	}
	if !strings.Contains(rt.Exporter().Snapshot(), "java_profiler_collector_target_status_orphaned_profiler_session 1") {
		t.Fatalf("expected orphaned status metric, got %q", rt.Exporter().Snapshot())
	}
}

func TestRuntimeScanOnceKeepsMissingOrDifferentMarkerProfilerConflict(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{name: "missing marker"},
		{
			name: "different marker",
			setup: func(t *testing.T, root string) {
				t.Helper()
				if err := profiler.WriteSessionMarker(root, 123, profiler.SessionMarker{
					CollectorID: "collector-2",
					PID:         123,
				}); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeRuntimeProcess(t, root, 123, true)
			uid := "11111111-2222-3333-4444-555555555555"
			if err := os.WriteFile(filepath.Join(root, "123", "cgroup"), []byte("0::/kubepods/pod"+uid+"/container"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.setup != nil {
				tc.setup(t, root)
			}
			attach := &recordingRuntimeAttach{}
			rt := NewCollector(Config{ProcRoot: root, CollectorID: "collector-1"})
			rt.profiler = profiler.NewRunner(profiler.Config{
				ProcRoot:     root,
				OwnerID:      "collector-1",
				TargetTmpDir: "/tmp/java-profiler",
			}, attach)
			rt.podSource = func(context.Context) map[string]podItem {
				return map[string]podItem{
					normalizeUID(uid): readyProfiledPod(uid),
				}
			}

			if err := rt.ScanOnce(context.Background()); err != nil {
				t.Fatal(err)
			}

			statuses := rt.Statuses()
			if len(statuses) != 1 {
				t.Fatalf("expected one status, got %d", len(statuses))
			}
			if statuses[0].Reason != domain.StatusReasonProfilerConflict {
				t.Fatalf("expected profiler conflict status, got %#v", statuses[0])
			}
			if len(attach.commands) != 0 {
				t.Fatalf("expected external conflict not to be stopped, got %#v", attach.commands)
			}
		})
	}
}

func TestCollectProfilesUsesLimitedConcurrency(t *testing.T) {
	release := make(chan struct{})
	started := make(chan int, 16)
	rt := &Runtime{
		profiler: &blockingProfileCollector{
			release: release,
			started: started,
		},
	}
	targets := make([]domain.TargetIdentity, 5)
	for i := range targets {
		targets[i] = domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod", ProcessID: i + 1}
	}

	done := make(chan struct{})
	var samples []profiling.ProfileSample
	var rawCount int
	var err error
	go func() {
		samples, rawCount, err = rt.collectProfiles(context.Background(), "batch-1", targets)
		close(done)
	}()

	seen := map[int]struct{}{}
	timeout := time.After(2 * time.Second)
	for len(seen) < maxConcurrentProfiles {
		select {
		case pid := <-started:
			seen[pid] = struct{}{}
		case <-timeout:
			t.Fatal("timed out waiting for concurrent collection start")
		}
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for collection to finish")
	}

	if err != nil {
		t.Fatalf("collectProfiles failed: %v", err)
	}
	if rawCount != len(targets) {
		t.Fatalf("rawCount = %d, want %d", rawCount, len(targets))
	}
	if len(samples) != len(targets) {
		t.Fatalf("samples = %d, want %d", len(samples), len(targets))
	}
	if got := rt.profiler.(*blockingProfileCollector).maxActive; got != maxConcurrentProfiles {
		t.Fatalf("max active = %d, want %d", got, maxConcurrentProfiles)
	}
}

type recordingRuntimeAttach struct {
	commands []recordedRuntimeAttach
}

type recordedRuntimeAttach struct {
	pid       int
	agentPath string
	args      string
}

func (a *recordingRuntimeAttach) LoadNativeAgent(_ context.Context, pid int, agentPath string, args string) error {
	a.commands = append(a.commands, recordedRuntimeAttach{pid: pid, agentPath: agentPath, args: args})
	return nil
}

func writeRuntimeProcess(t *testing.T, root string, pid int, conflict bool) {
	t.Helper()
	pidString := strconv.Itoa(pid)
	pidDir := filepath.Join(root, pidString)
	if err := os.Mkdir(pidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  0 0 0 0 0 0 0 0 0 0\nbtime 1000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "cmdline"), []byte("java\x00-jar\x00app.jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pidDir, "stat"), []byte(pidString+" (java) S 1 1 1 0 -1 4194560 0 0 0 0 0 0 0 0 20 0 1 0 200 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"), 0o644); err != nil {
		t.Fatal(err)
	}
	maps := "/usr/lib/jvm/java-17/lib/server/libjvm.so hotspot\n"
	if conflict {
		maps += "/tmp/java-profiler/libasyncProfiler.so\n"
	}
	if err := os.WriteFile(filepath.Join(pidDir, "maps"), []byte(maps), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readyProfiledPod(uid string) podItem {
	return podItem{
		Metadata: podMetadata{
			Name:      "demo",
			Namespace: "default",
			UID:       uid,
			Labels: map[string]string{
				"app.kubernetes.io/name": "demo",
			},
			Annotations: map[string]string{
				policy.AnnotationProfileMode: "continuous",
			},
			CreationTimestamp: timeWrapper{Time: time.Unix(1, 0).UTC()},
		},
		Spec: podSpec{
			NodeName: "node-a",
			Containers: []podContainer{
				{Name: "app"},
			},
		},
		Status: podStatus{
			Conditions: []podCondition{
				{Type: "Ready", Status: "True"},
			},
		},
	}
}

type blockingProfileCollector struct {
	release   <-chan struct{}
	started   chan<- int
	mu        sync.Mutex
	active    int
	maxActive int
}

func (c *blockingProfileCollector) Collect(_ context.Context, _ string, target domain.TargetIdentity) (profiler.CollectionResult, error) {
	c.mu.Lock()
	c.active++
	if c.active > c.maxActive {
		c.maxActive = c.active
	}
	c.mu.Unlock()
	c.started <- target.ProcessID
	<-c.release
	c.mu.Lock()
	c.active--
	c.mu.Unlock()
	return profiler.CollectionResult{
		Samples: []profiling.ProfileSample{{
			BatchID:     "batch-1",
			Target:      target,
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(1, 0),
			EndedAt:     time.Unix(2, 0),
			StackID:     "stack",
			Value:       1,
		}},
		RawSampleCount: 1,
	}, nil
}

func (c *blockingProfileCollector) HasSession(domain.TargetIdentity) bool { return false }

func (c *blockingProfileCollector) RecoverConflict(context.Context, domain.TargetIdentity) (profiler.ConflictRecovery, error) {
	return profiler.ConflictRecovery{}, nil
}
