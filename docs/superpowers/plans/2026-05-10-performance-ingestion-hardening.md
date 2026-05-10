# Performance Ingestion Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Java profiler ingestion path survive real Kubernetes performance tests: bounded uploads, collector-side aggregation, visible drops/truncation, queryable self/total data, async-profiler session recovery, and an acceptance scenario that proves CPU, allocation, and lock profiles are real data instead of empty success paths.

**Architecture:** Collector normalizes async-profiler/JFR events into profile samples, aggregates duplicate stacks per target/profile/window, applies explicit limits, uploads bounded batches with metadata, backend records ingestion health and stores aggregated samples in ClickHouse, UI queries aggregate data for Top Table and Flame Graph. Raw profile artifacts remain optional and short-lived. ClickHouse is treated as a constrained shared dependency, not an infinite row sink.

**Tech Stack:** Go collector/backend, async-profiler/JFR, ClickHouse, React/TypeScript UI, Kubernetes acceptance scripts, GitHub Actions profile demo image workflow.

---

## Required References

Read these files before changing code:

- `docs/architecture/performance-ingestion-architecture-review.md`
- `docs/operations/real-profiling-acceptance-standard.md`
- `docs/research/pyroscope-profile-ui-study.md`
- `docs/research/coroot-node-agent-java-agent.md`
- `collector/runtime/runtime.go`
- `collector/internal/jfr/normalizer.go`
- `collector/internal/pipeline/profile_batcher.go`
- `backend/internal/app/ingest_profile_batch.go`
- `backend/internal/httpapi/ingest_handlers.go`
- `backend/internal/clickhouse/sql_repository.go`
- `backend/internal/clickhouse/ingestion_repository.go`
- `web/src/features/cpu/hot-code-view.tsx`
- `web/src/features/ingestion/ingestion-health-view.tsx`

## File Structure

New files:

- `collector/internal/jfr/aggregate.go` - stack aggregation by target, profile type, time window, and stack id.
- `collector/internal/jfr/aggregate_test.go` - aggregation unit tests.
- `collector/internal/pipeline/profile_limits.go` - bounded profile batch limit logic.
- `collector/internal/pipeline/profile_limits_test.go` - truncation and metadata tests.
- `collector/internal/profiler/session_marker.go` - async-profiler ownership marker read/write logic.
- `collector/internal/profiler/session_marker_test.go` - stale-owned and external-conflict tests.

Modified files:

- `contracts/profiling/types.go` - add profile batch metadata contract.
- `collector/internal/pipeline/profile_batcher.go` - serialize metadata with every profile batch.
- `collector/runtime/runtime.go` - aggregate, bound, split, and upload profiles with visible metadata.
- `collector/internal/jfr/normalizer.go` - return aggregated samples from normalized windows.
- `backend/internal/app/ingest_profile_batch.go` - persist profile batch metadata and reject impossible batches.
- `backend/internal/app/ingest_profile_batch_test.go` - ingestion metadata and retry tests.
- `backend/internal/httpapi/ingest_handlers.go` - distinguish 413 body-too-large from invalid JSON.
- `backend/internal/httpapi/ingest_handlers_test.go` - body limit and sample limit tests.
- `backend/internal/clickhouse/ingestion_repository.go` - store truncation/drop counters.
- `backend/internal/clickhouse/sql_repository.go` - insert/query bounded aggregate samples and ingestion metadata.
- `backend/internal/clickhouse/001_initial_profile_schema.sql` - add metadata columns to `java_profiler_ingestion_batches`.
- `backend/internal/clickhouse/schema.go` - keep schema creation order unchanged.
- `backend/internal/app/query_ingestion_health.go` - expose dropped/truncated counters.
- `backend/internal/app/query_top_stacks.go` - ensure self/total semantics are computed from aggregates.
- `backend/internal/httpapi/query_handlers.go` - expose top-table data if not already wired.
- `web/src/api/types.ts` - add ingestion metadata and top table self/total fields.
- `web/src/api/client.ts` - fetch top table from backend instead of inferring from flame graph only.
- `web/src/features/cpu/hot-code-view.tsx` - keep Pyroscope-style Top Table / Flame Graph / Both interaction without source-code panel.
- `web/src/features/ingestion/ingestion-health-view.tsx` - make drops, retries, and truncation visible.
- `scripts/real-acceptance.sh` - add high-volume profile ingestion acceptance mode.
- `docs/operations/real-profiling-acceptance-standard.md` - add bounded-ingestion acceptance criteria.
- `docs/architecture/performance-ingestion-architecture-review.md` - mark implemented decisions.

## Acceptance Gates

The implementation is not complete until all gates pass:

- CPU profile has non-empty Java frames and Top Table ranks by `total` and `self`.
- Allocation profile has non-empty samples under real load.
- Lock profile has non-empty samples produced by concurrent contention.
- Profile uploads never exceed backend body limits.
- Collector reports dropped/truncated counts when limits are hit.
- Backend ingestion health shows accepted, retryable, rejected, dropped, and truncated states.
- ClickHouse remains responsive during high-volume profile generation.
- UI search filters Top Table and Flame Graph visibly.
- Flame Graph shows stack hierarchy and tooltip values; Top Table shows self and total.
- No source-code panel is shown in the UI.
- Async-profiler conflict status distinguishes collector-owned stale state from external profiler usage.

## Task 1: Add Profile Batch Metadata Contract

**Purpose:** Every uploaded profile batch must carry enough metadata to prove whether the collector aggregated, split, truncated, or dropped data.

- [ ] Add this type to `contracts/profiling/types.go`:

```go
type ProfileBatchMetadata struct {
	WindowRawSampleCount        int  `json:"window_raw_sample_count"`
	WindowAggregatedSampleCount int  `json:"window_aggregated_sample_count"`
	BatchSampleCount            int  `json:"batch_sample_count"`
	DroppedSampleCount          int  `json:"dropped_sample_count"`
	DroppedStackCount           int  `json:"dropped_stack_count"`
	Truncated                   bool `json:"truncated"`
	PartIndex                   int  `json:"part_index"`
	PartCount                   int  `json:"part_count"`
}
```

- [ ] Change `collector/internal/pipeline/profile_batcher.go` so `ProfileBatchPayload` includes:

```go
Metadata profiling.ProfileBatchMetadata `json:"metadata"`
```

- [ ] Change `BuildProfileBatch` to accept metadata:

```go
func BuildProfileBatch(batchID, collectorID string, samples []profiling.ProfileSample, metadata profiling.ProfileBatchMetadata) ([]byte, error)
```

- [ ] Add `collector/internal/pipeline/profile_batcher_test.go` coverage that validates metadata is present in JSON:

```go
func TestBuildProfileBatchIncludesMetadata(t *testing.T) {
	payload, err := BuildProfileBatch(
		"batch-a",
		"collector-a",
		nil,
		profiling.ProfileBatchMetadata{
			WindowRawSampleCount:        100,
			WindowAggregatedSampleCount: 12,
			BatchSampleCount:            10,
			DroppedSampleCount:          2,
			DroppedStackCount:           1,
			Truncated:                   true,
			PartIndex:                   1,
			PartCount:                   2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ProfileBatchPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Metadata.WindowRawSampleCount != 100 {
		t.Fatalf("raw sample count = %d", decoded.Metadata.WindowRawSampleCount)
	}
	if !decoded.Metadata.Truncated {
		t.Fatalf("metadata should mark the batch as truncated")
	}
}
```

- [ ] Update all `BuildProfileBatch` call sites.

- [ ] Run:

```bash
go test ./collector/internal/pipeline ./contracts/...
```

- [ ] Commit:

```bash
git add contracts/profiling/types.go collector/internal/pipeline/profile_batcher.go collector/internal/pipeline/profile_batcher_test.go
git commit -m "feat: add profile batch ingestion metadata"
```

## Task 2: Aggregate Profile Samples Before Upload

**Purpose:** Reduce the upload shape from one row per profiler event to one row per unique stack per target/profile/window. This directly addresses oversized uploads and ClickHouse OOM risk.

- [ ] Create `collector/internal/jfr/aggregate.go`:

```go
package jfr

import (
	"sort"
	"strings"

	"github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func AggregateSamples(samples []profiling.ProfileSample) []profiling.ProfileSample {
	if len(samples) == 0 {
		return nil
	}

	type key struct {
		namespace   string
		pod         string
		container   string
		service     string
		node        string
		cluster     string
		profileType domain.ProfileType
		startedAt   int64
		endedAt     int64
		stackID     string
	}

	aggregated := make(map[key]profiling.ProfileSample, len(samples))
	for _, sample := range samples {
		k := key{
			namespace:   sample.Target.Namespace,
			pod:         sample.Target.Pod,
			container:   sample.Target.Container,
			service:     sample.Target.Service,
			node:        sample.Target.Node,
			cluster:     sample.Target.Cluster,
			profileType: sample.ProfileType,
			startedAt:   sample.StartedAt.UnixNano(),
			endedAt:     sample.EndedAt.UnixNano(),
			stackID:     sample.StackID,
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

	result := make([]profiling.ProfileSample, 0, len(aggregated))
	for _, sample := range aggregated {
		result = append(result, sample)
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.Join(result[i].Frames, "\x00")
		right := strings.Join(result[j].Frames, "\x00")
		if result[i].ProfileType != result[j].ProfileType {
			return result[i].ProfileType < result[j].ProfileType
		}
		if result[i].Value != result[j].Value {
			return result[i].Value > result[j].Value
		}
		return left < right
	})
	return result
}
```

- [ ] Add `collector/internal/jfr/aggregate_test.go` with one duplicate-stack test and one different-profile-type test:

```go
func TestAggregateSamplesSumsDuplicateStackValues(t *testing.T) {
	now := time.Unix(1, 0)
	target := domain.TargetIdentity{Namespace: "java-profiler-qa", Pod: "demo", Service: "jdk17-http-demo"}
	samples := []profiling.ProfileSample{
		{Target: target, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 3},
		{Target: target, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"root", "Demo.burnCpu:188"}, Value: 5},
	}

	got := AggregateSamples(samples)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Value != 8 {
		t.Fatalf("value = %d", got[0].Value)
	}
}
```

- [ ] Change `collector/internal/jfr/normalizer.go` so `NormalizeWindow` returns `AggregateSamples(samples)` after all event conversion.

- [ ] Extend `collector/internal/jfr/normalizer_test.go` to assert duplicate JFR events with the same stack produce one sample with summed `Value`.

- [ ] Run:

```bash
go test ./collector/internal/jfr
```

- [ ] Commit:

```bash
git add collector/internal/jfr/aggregate.go collector/internal/jfr/aggregate_test.go collector/internal/jfr/normalizer.go collector/internal/jfr/normalizer_test.go
git commit -m "feat: aggregate profile stacks before upload"
```

## Task 3: Enforce Collector-Side Profile Limits

**Purpose:** Make upload volume deterministic. The collector must not depend on backend failures to discover that a batch is too large.

- [ ] Create `collector/internal/pipeline/profile_limits.go`:

```go
package pipeline

import "github.com/koolay/java-profiler/contracts/profiling"

type ProfileBatchLimits struct {
	MaxSamplesPerWindow int
	MaxSamplesPerBatch  int
}

func DefaultProfileBatchLimits() ProfileBatchLimits {
	return ProfileBatchLimits{
		MaxSamplesPerWindow: 50_000,
		MaxSamplesPerBatch:  10_000,
	}
}

func BoundProfileSamples(samples []profiling.ProfileSample, limits ProfileBatchLimits) ([]profiling.ProfileSample, profiling.ProfileBatchMetadata) {
	metadata := profiling.ProfileBatchMetadata{
		WindowRawSampleCount:        len(samples),
		WindowAggregatedSampleCount: len(samples),
	}
	if limits.MaxSamplesPerWindow <= 0 || len(samples) <= limits.MaxSamplesPerWindow {
		return samples, metadata
	}

	bounded := make([]profiling.ProfileSample, limits.MaxSamplesPerWindow)
	copy(bounded, samples[:limits.MaxSamplesPerWindow])
	metadata.DroppedSampleCount = len(samples) - limits.MaxSamplesPerWindow
	metadata.DroppedStackCount = metadata.DroppedSampleCount
	metadata.Truncated = true
	return bounded, metadata
}

func BatchMetadataForPart(base profiling.ProfileBatchMetadata, partIndex, partCount, batchSampleCount int) profiling.ProfileBatchMetadata {
	base.PartIndex = partIndex
	base.PartCount = partCount
	base.BatchSampleCount = batchSampleCount
	return base
}
```

- [ ] Add `collector/internal/pipeline/profile_limits_test.go`:

```go
func TestBoundProfileSamplesMarksTruncation(t *testing.T) {
	samples := []profiling.ProfileSample{
		{StackID: "a", Value: 1},
		{StackID: "b", Value: 1},
		{StackID: "c", Value: 1},
	}
	got, meta := BoundProfileSamples(samples, ProfileBatchLimits{MaxSamplesPerWindow: 2, MaxSamplesPerBatch: 1})
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if !meta.Truncated {
		t.Fatalf("expected truncated metadata")
	}
	if meta.DroppedSampleCount != 1 {
		t.Fatalf("dropped sample count = %d", meta.DroppedSampleCount)
	}
}
```

- [ ] Modify `collector/runtime/runtime.go`:

  - Add `profileLimits pipeline.ProfileBatchLimits` to `Runtime`.
  - Set `profileLimits: pipeline.DefaultProfileBatchLimits()` in `New`.
  - In `uploadProfileSamples`, call `pipeline.BoundProfileSamples` before splitting.
  - Calculate `partCount` with `(len(samples)+maxPerBatch-1)/maxPerBatch`.
  - Pass `pipeline.BatchMetadataForPart(metadata, partIndex, partCount, len(chunk))` to `BuildProfileBatch`.

- [ ] Keep `maxProfileSamplesPerBatch` behavior but drive it from `profileLimits.MaxSamplesPerBatch`.

- [ ] Run:

```bash
go test ./collector/runtime ./collector/internal/pipeline
```

- [ ] Commit:

```bash
git add collector/runtime/runtime.go collector/internal/pipeline/profile_limits.go collector/internal/pipeline/profile_limits_test.go
git commit -m "feat: bound profile uploads in collector"
```

## Task 4: Persist and Expose Ingestion Metadata

**Purpose:** Operators must see when data was truncated, dropped, retried, or rejected. A green UI with hidden loss is not acceptable.

- [ ] Extend `backend/internal/app/ingest_profile_batch.go` request type:

```go
type ProfileBatchRequest struct {
	BatchID     string                         `json:"batch_id"`
	CollectorID string                         `json:"collector_id"`
	ReceivedAt  time.Time                      `json:"received_at"`
	Metadata    profiling.ProfileBatchMetadata `json:"metadata"`
	Samples     []clickhouse.ProfileSample     `json:"samples"`
}
```

- [ ] Add these fields to `backend/internal/clickhouse/ingestion_repository.go` `IngestionBatch`:

```go
RawSampleCount        int
AggregatedSampleCount int
BatchSampleCount      int
DroppedSampleCount    int
DroppedStackCount     int
Truncated             bool
```

- [ ] Update the SQL ingestion table schema to include:

```sql
raw_sample_count UInt64 DEFAULT 0,
aggregated_sample_count UInt64 DEFAULT 0,
batch_sample_count UInt64 DEFAULT 0,
dropped_sample_count UInt64 DEFAULT 0,
dropped_stack_count UInt64 DEFAULT 0,
truncated UInt8 DEFAULT 0
```

- [ ] When `ProfileBatchIngestor.Ingest` records `claimed`, `accepted`, `rejected`, or `retryable`, copy request metadata into the `IngestionBatch`.

- [ ] Add backend test coverage in `backend/internal/app/ingest_profile_batch_test.go`:

```go
func TestProfileBatchIngestRecordsMetadata(t *testing.T) {
	ingestion := clickhouse.NewIngestionRepository()
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: ingestion,
	}

	result, err := ingestor.Ingest(context.Background(), ProfileBatchRequest{
		BatchID:     "batch-meta",
		CollectorID: "collector-a",
		ReceivedAt:  time.Unix(1, 0),
		Metadata: profiling.ProfileBatchMetadata{
			WindowRawSampleCount:        100,
			WindowAggregatedSampleCount: 20,
			BatchSampleCount:            10,
			DroppedSampleCount:          5,
			DroppedStackCount:           4,
			Truncated:                   true,
		},
		Samples: []clickhouse.ProfileSample{{
			BatchID:     "batch-meta",
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(100, 0),
			EndedAt:     time.Unix(160, 0),
			StackID:     "stack-1",
			Frames:      []string{"root", "Demo.burnCpu:188"},
			Value:       10,
		}},
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("status = %s", result.Status)
	}

	batches, err := ingestion.ListIngestionBatches(context.Background(), clickhouse.IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) == 0 {
		t.Fatalf("expected ingestion batch")
	}
	if batches[0].DroppedSampleCount != 5 {
		t.Fatalf("dropped sample count = %d", batches[0].DroppedSampleCount)
	}
	if !batches[0].Truncated {
		t.Fatalf("expected truncated batch")
	}
}
```

- [ ] Modify `backend/internal/httpapi/ingest_handlers.go` to return `413 Request Entity Too Large` for `*http.MaxBytesError`:

```go
var maxBytesErr *http.MaxBytesError
if errors.As(err, &maxBytesErr) {
	http.Error(w, "profile batch too large", http.StatusRequestEntityTooLarge)
	return
}
```

- [ ] Add `backend/internal/httpapi/ingest_handlers_test.go` coverage for oversized profile upload.

- [ ] Update `backend/internal/app/query_ingestion_health.go` and `web/src/api/types.ts` to expose totals:

```go
DroppedSamples  int `json:"dropped_samples"`
DroppedStacks   int `json:"dropped_stacks"`
TruncatedBatches int `json:"truncated_batches"`
```

- [ ] Update `web/src/features/ingestion/ingestion-health-view.tsx` so the Ingestion tab shows:

  - accepted batches
  - retryable batches
  - rejected batches
  - dropped samples
  - dropped stacks
  - truncated batches
  - latest retry/rejection message

- [ ] Run:

```bash
go test ./backend/internal/app ./backend/internal/httpapi ./backend/internal/clickhouse
cd web && npm test -- --run
```

- [ ] Commit:

```bash
git add backend/internal/app backend/internal/httpapi backend/internal/clickhouse web/src/api web/src/features/ingestion
git commit -m "feat: expose bounded ingestion health"
```

## Task 5: Harden Top Table and Flame Graph Query Semantics

**Purpose:** The UI must support real performance localization. Top Table answers "where is cost owned"; Flame Graph answers "which sampled stack paths lead there." This mirrors Pyroscope's core interaction without adding Pyroscope as a backend dependency.

- [ ] Replace `backend/internal/app/query_top_stacks.go` with a query layer that derives Top Table rows from `ProfileQueryStore.QuerySamples`, groups self by leaf frame, and groups total by every frame in each stack.

- [ ] Top Table response rows must include:

```go
type TopStackRow struct {
	Symbol       string `json:"symbol"`
	Location     string `json:"location"`
	ProfileType  string `json:"profile_type"`
	Self         uint64 `json:"self"`
	Total        uint64 `json:"total"`
	SelfPercent  string `json:"self_percent"`
	TotalPercent string `json:"total_percent"`
}
```

- [ ] Self calculation rule:

  - The leaf frame of each stack receives `self += sample.Value`.
  - Every frame in the same stack receives `total += sample.Value`.
  - Java application frames are not hidden from Top Table.
  - JVM/native frames can appear in Flame Graph but must not displace Java application rows from Top Table when Java frames exist in the same stack.

- [ ] Add these helpers to `backend/internal/app/query_top_stacks.go`:

```go
// Add fmt to the existing imports. Keep context and sort.
import (
	"context"
	"fmt"
	"sort"
)

func rankTopStacks(samples []clickhouse.ProfileSample) []TopStackRow {
	totalProfileValue := uint64(0)
	rowsBySymbol := map[string]TopStackRow{}
	for _, sample := range samples {
		totalProfileValue += sample.Value
		if len(sample.Frames) == 0 {
			continue
		}
		leaf := sample.Frames[len(sample.Frames)-1]
		for _, frame := range sample.Frames {
			if frame == "root" {
				continue
			}
			row := rowsBySymbol[frame]
			row.Symbol = frame
			row.Location = frame
			row.ProfileType = sample.ProfileType.String()
			row.Total += sample.Value
			if frame == leaf {
				row.Self += sample.Value
			}
			rowsBySymbol[frame] = row
		}
	}
	rows := make([]TopStackRow, 0, len(rowsBySymbol))
	for _, row := range rowsBySymbol {
		row.SelfPercent = formatPercent(row.Self, totalProfileValue)
		row.TotalPercent = formatPercent(row.Total, totalProfileValue)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		if rows[i].Self != rows[j].Self {
			return rows[i].Self > rows[j].Self
		}
		return rows[i].Symbol < rows[j].Symbol
	})
	return rows
}

func formatPercent(value uint64, total uint64) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(value)*100/float64(total))
}
```

- [ ] Add backend test coverage to `backend/internal/app/query_top_stacks_test.go`:

```go
// Add strings to the test imports.
import "strings"

func findTopStackRow(rows []TopStackRow, contains string) TopStackRow {
	for _, row := range rows {
		if strings.Contains(row.Symbol, contains) {
			return row
		}
	}
	return TopStackRow{}
}

func TestTopStacksSeparatesSelfAndTotal(t *testing.T) {
	samples := []clickhouse.ProfileSample{
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "Demo.handleWork:93", "Demo.burnCpu:188"}, Value: 8},
		{ProfileType: domain.ProfileTypeCPU, Frames: []string{"root", "Demo.handleWork:93", "Demo.writeJson:232"}, Value: 2},
	}

	rows := rankTopStacks(samples)
	handle := findTopStackRow(rows, "Demo.handleWork")
	burn := findTopStackRow(rows, "Demo.burnCpu")

	if handle.Total != 10 || handle.Self != 0 {
		t.Fatalf("handle self/total = %d/%d", handle.Self, handle.Total)
	}
	if burn.Total != 8 || burn.Self != 8 {
		t.Fatalf("burn self/total = %d/%d", burn.Self, burn.Total)
	}
}
```

- [ ] Update `web/src/features/cpu/hot-code-view.tsx`:

  - Remove source-code context rendering.
  - Keep three view modes: `Top Table`, `Flame Graph`, `Both`.
  - In Top Table mode, display `Symbol`, `Self`, and `Total`.
  - In Both mode, keep Top Table left and Flame Graph right.
  - Clicking a Top Table row filters/highlights corresponding Flame Graph frames.
  - Search input filters both Top Table rows and Flame Graph frame labels.
  - Back button restores the previous focused flame root.
  - Reset clears search, selection, and focused root.

- [ ] Keep explanatory copy short and operational:

```text
Top table ranks Java symbols by self and total CPU samples. The flame graph shows sampled stack context, not source call order.
```

- [ ] Add these UI tests to `web/src/features/cpu/hot-code-view.test.tsx`:

```tsx
test("search filters top table rows and flame graph frames", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  fireEvent.change(screen.getByPlaceholderText("Search frame"), { target: { value: "handleWork" } });

  const table = screen.getByRole("region", { name: "Top table" });
  expect(within(table).getByRole("row", { name: /DemoHttpService\.handleWork/ })).toBeInTheDocument();
  expect(within(table).queryByRole("row", { name: /CheckoutService\.priceCart/ })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /DemoHttpService\.handleWork:93/ })).toHaveClass("flame-row-match");
});

test("reset clears search and selected frame", () => {
  render(<HotCodeView root={root} metadata={{ partial: false }} />);

  fireEvent.change(screen.getByPlaceholderText("Search frame"), { target: { value: "burnCpu" } });
  fireEvent.click(screen.getByRole("button", { name: /DemoHttpService\.burnCpu:188/ }));
  fireEvent.click(screen.getByRole("button", { name: "Reset" }));

  expect(screen.getByPlaceholderText("Search frame")).toHaveValue("");
  expect(screen.queryByText("SELECTED FRAME")).not.toBeInTheDocument();
  expect(screen.getByRole("row", { name: /CheckoutService\.priceCart/ })).toBeInTheDocument();
});
```

- [ ] Run:

```bash
go test ./backend/internal/app ./backend/internal/httpapi
cd web && npm test -- --run
```

- [ ] Commit:

```bash
git add backend/internal/app backend/internal/httpapi web/src/api web/src/features/cpu
git commit -m "feat: align profile UI with self total analysis"
```

## Task 6: Add Async-Profiler Session Ownership Recovery

**Purpose:** A collector restart must not leave the target JVM permanently unprofilable. An external profiler conflict must remain protected.

- [ ] Create `collector/internal/profiler/session_marker.go`:

```go
package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type SessionMarker struct {
	CollectorID string    `json:"collector_id"`
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"started_at"`
	LibraryPath string    `json:"library_path"`
}

func markerPath(procRoot string, pid int) string {
	return filepath.Join(procRoot, strconv.Itoa(pid), "root", "tmp", "java-profiler-session.json")
}

func WriteSessionMarker(procRoot string, pid int, marker SessionMarker) error {
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return os.WriteFile(markerPath(procRoot, pid), data, 0o600)
}

func ReadSessionMarker(procRoot string, pid int) (SessionMarker, error) {
	data, err := os.ReadFile(markerPath(procRoot, pid))
	if err != nil {
		return SessionMarker{}, err
	}
	var marker SessionMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return SessionMarker{}, err
	}
	return marker, nil
}
```

- [ ] Add `collector/internal/profiler/session_marker_test.go` that writes and reads a marker under a temporary proc root.

- [ ] Add `OwnerID string` to profiler config and pass `Runtime.collectorID`.

- [ ] Write the marker before starting async-profiler and remove it after a clean stop.

- [ ] Change target status logic in `collector/runtime/runtime.go`:

  - If async-profiler is loaded and marker owner equals this collector, status reason is `orphaned_profiler_session`.
  - If async-profiler is loaded and marker owner is different or marker is missing, status reason is `profiler_conflict`.
  - If owned orphan recovery succeeds, next scan may profile the target again.

- [ ] Add the new status reason constant in `domain/types.go` next to the existing `StatusReasonProfilerConflict` constant:

```go
StatusReasonOrphanedProfilerSession StatusReason = "orphaned_profiler_session"
```

- [ ] Add tests to cover:

  - owned stale marker can be recovered
  - missing marker remains external conflict
  - different collector id remains external conflict

- [ ] Run:

```bash
go test ./collector/internal/profiler ./collector/runtime ./contracts/...
```

- [ ] Commit:

```bash
git add collector/internal/profiler collector/runtime contracts domain
git commit -m "feat: recover owned async-profiler sessions"
```

## Task 7: Add High-Volume Kubernetes Acceptance

**Purpose:** Prove the ingestion design holds under realistic pressure: CPU, allocation, lock contention, repeated batches, ClickHouse pressure, and UI inspection.

- [ ] Update `scripts/real-acceptance.sh` with an opt-in high-volume mode:

```bash
JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME="${JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME:-0}"
JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME_SECONDS="${JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME_SECONDS:-180}"
JAVA_PROFILER_ACCEPTANCE_CONCURRENCY="${JAVA_PROFILER_ACCEPTANCE_CONCURRENCY:-32}"
```

- [ ] In high-volume mode, drive the JDK17 demo service with:

```bash
for mode in cpu alloc lock; do
  seq 1 "$JAVA_PROFILER_ACCEPTANCE_CONCURRENCY" | xargs -P "$JAVA_PROFILER_ACCEPTANCE_CONCURRENCY" -I{} \
    sh -c "end=\$((SECONDS + $JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME_SECONDS)); while [ \$SECONDS -lt \$end ]; do curl -fsS \"$DEMO_BASE_URL/work?mode=$mode&durationMs=1000\" >/dev/null; done"
done
```

- [ ] Add checks that fail when any required profile type is empty:

```bash
require_profile_type cpu
require_profile_type alloc
require_profile_type lock
```

- [ ] Add checks that fail when backend ingestion health hides data loss:

```bash
assert_json_number_at_least "$INGESTION_HEALTH_JSON" ".totals.accepted" 1
assert_json_field_present "$INGESTION_HEALTH_JSON" ".totals.dropped_samples"
assert_json_field_present "$INGESTION_HEALTH_JSON" ".totals.truncated_batches"
```

- [ ] Add checks that ClickHouse remains responsive:

```bash
kubectl -n java-profiler-qa exec deploy/java-profiler-clickhouse -- clickhouse-client --query "SELECT 1"
```

- [ ] Update `docs/operations/real-profiling-acceptance-standard.md` with a new section:

```markdown
### Bounded Ingestion Acceptance

The run is accepted only when profile data is real and bounded:

- CPU, allocation, and lock profiles each contain non-empty samples.
- Ingestion health shows accepted profile batches.
- Drop and truncation counters are visible even when they are zero.
- Oversized profile batches are split or truncated by the collector before upload.
- Backend body-limit failures return HTTP 413 and are visible as rejected ingestion.
- ClickHouse answers a health query after the high-volume run.
```

- [ ] Run a real Kubernetes test:

```bash
export KUBECONFIG=$HOME/backup/localk8s.yaml
JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME=1 \
JAVA_PROFILER_ACCEPTANCE_HIGH_VOLUME_SECONDS=180 \
JAVA_PROFILER_ACCEPTANCE_CONCURRENCY=32 \
./scripts/real-acceptance.sh
```

- [ ] Capture the local UI URL and verify manually or through browser automation:

```bash
kubectl -n java-profiler-qa port-forward svc/java-profiler-web 18181:80
```

Expected URL:

```text
http://127.0.0.1:18181
```

- [ ] Commit:

```bash
git add scripts/real-acceptance.sh docs/operations/real-profiling-acceptance-standard.md
git commit -m "test: add high volume profiling acceptance"
```

## Task 8: Final Verification and Review

- [ ] Run full Go tests:

```bash
go test ./...
```

- [ ] Run web tests:

```bash
cd web && npm test -- --run
```

- [ ] Run the real Kubernetes acceptance command from Task 7.

- [ ] Open `http://127.0.0.1:18181` and verify:

  - default filter targets Java pods only
  - CPU Top Table contains Java application rows
  - Top Table has non-zero `total`
  - Flame Graph frame widths reflect total resource share
  - search changes visible rows/frames
  - Back and Reset work
  - Memory/Allocation tab is non-empty
  - Locks tab is non-empty under contention
  - Ingestion tab shows accepted and bounded metadata

- [ ] Run code review:

```bash
git diff --stat HEAD~7..HEAD
git diff HEAD~7..HEAD
```

- [ ] Reject the branch if any of these are true:

  - profile uploads can grow without collector-side caps
  - UI shows runtime/native frames as the only actionable result when Java frames exist
  - ingestion health cannot show hidden drops/truncation
  - real acceptance passes without CPU, allocation, and lock samples
  - source-code panel returns to the CPU profile UI

## Risk Controls

- Do not add Pyroscope, Parca, or Grafana as required backend dependencies.
- Do not expand scope beyond Java services on Kubernetes.
- Do not store unbounded raw profile rows in ClickHouse.
- Do not rely on large ClickHouse memory limits as the fix.
- Do not mark Kubernetes acceptance as passing from HTTP success alone.
- Do not hide profile data loss behind successful ingestion status.

## Implementation Notes

- Keep raw JFR artifacts optional. Aggregated samples are the primary query path.
- Preserve 7-day-or-less retention assumptions.
- Keep Prometheus as metric storage only if already configured; profiling data remains in ClickHouse.
- Keep profiler conflict recovery conservative. Unknown ownership is an external conflict.
- Treat the UI as an analysis tool: Top Table for ranking, Flame Graph for stack context, Ingestion for trust in data.

## Self-Review

This plan directly addresses the failures captured in `performance-ingestion-architecture-review.md`: oversized uploads, ClickHouse OOM pressure, hidden ingestion loss, stale async-profiler conflicts, and UI paths that previously looked successful without useful performance data. The tasks are ordered so contracts land first, collector volume is bounded before backend/UI expansion, and real Kubernetes acceptance closes the loop.

The highest-risk implementation point is schema compatibility for existing ClickHouse tables. If local environments already have old tables, the implementation must either add an idempotent migration or document the reset command used by the test environment. Do not skip this; otherwise the code can pass unit tests and fail in real acceptance.

The second highest-risk point is Top Table self/total semantics. Tests must include stacks where Java methods have `total > 0` and `self = 0`, because that is the expected case when CPU lands in callees or runtime/native frames.
