package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koolay/java-profiler/collector/internal/pipeline"
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
