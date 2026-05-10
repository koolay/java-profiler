package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

func TestProfileBatchIngestorAcceptsAndDeduplicates(t *testing.T) {
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	req := ProfileBatchRequest{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		Samples: []clickhouse.ProfileSample{{
			BatchID:     "batch-1",
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(100, 0),
			EndedAt:     time.Unix(160, 0),
			StackID:     "stack-1",
			Value:       10,
		}},
	}
	result, err := ingestor.Ingest(context.Background(), req)
	if err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected accepted, got %+v err=%v", result, err)
	}
	result, err = ingestor.Ingest(context.Background(), req)
	if err != nil || result.Status != clickhouse.IngestionDuplicate {
		t.Fatalf("expected duplicate, got %+v err=%v", result, err)
	}
	profiles := ingestor.Profiles.(appProfileQueryStore)
	samples, err := profiles.QuerySamples(context.Background(), clickhouse.ProfileQuery{Namespace: "prod", Service: "checkout"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("duplicate upload should not inflate samples: %+v", samples)
	}
}

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

type appProfileQueryStore interface {
	QuerySamples(context.Context, clickhouse.ProfileQuery) ([]clickhouse.ProfileSample, error)
}

func TestProfileBatchIngestorRejectsSameBatchDifferentPayload(t *testing.T) {
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	base := ProfileBatchRequest{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		Samples: []clickhouse.ProfileSample{{
			BatchID:     "batch-1",
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(100, 0),
			EndedAt:     time.Unix(160, 0),
			StackID:     "stack-1",
			Frames:      []string{"A"},
			Value:       10,
		}},
	}
	if result, err := ingestor.Ingest(context.Background(), base); err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected accepted, got %+v err=%v", result, err)
	}
	changed := base
	changed.Samples = append([]clickhouse.ProfileSample(nil), base.Samples...)
	changed.Samples[0].Value = 99
	result, err := ingestor.Ingest(context.Background(), changed)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected reused batch id with changed payload to be rejected, got %+v", result)
	}
	profiles := ingestor.Profiles.(appProfileQueryStore)
	samples, err := profiles.QuerySamples(context.Background(), clickhouse.ProfileQuery{Namespace: "prod", Service: "checkout"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(samples) != 1 || samples[0].Value != 10 {
		t.Fatalf("conflicting batch should not write payload rows: %+v", samples)
	}
}

func TestProfileBatchIngestorRejectsSameBatchDifferentMetadata(t *testing.T) {
	ingestion := clickhouse.NewIngestionRepository()
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: ingestion,
	}
	base := ProfileBatchRequest{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		Metadata: profiling.ProfileBatchMetadata{
			WindowRawSampleCount: 10,
			BatchSampleCount:     1,
		},
		Samples: []clickhouse.ProfileSample{{
			BatchID:     "batch-1",
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(100, 0),
			EndedAt:     time.Unix(160, 0),
			StackID:     "stack-1",
			Frames:      []string{"A"},
			Value:       10,
		}},
	}
	if result, err := ingestor.Ingest(context.Background(), base); err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected accepted, got %+v err=%v", result, err)
	}
	changed := base
	changed.Metadata.DroppedSampleCount = 5
	changed.Metadata.Truncated = true
	result, err := ingestor.Ingest(context.Background(), changed)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected changed metadata to reject reused batch id, got %+v", result)
	}
	batches, err := ingestion.ListIngestionBatches(context.Background(), clickhouse.IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected one final ingestion state, got %+v", batches)
	}
	if batches[0].Status != clickhouse.IngestionRejected {
		t.Fatalf("expected final rejected state, got %+v", batches[0])
	}
	if batches[0].Message != "batch id reused with different payload" {
		t.Fatalf("expected conflict message, got %+v", batches[0])
	}
}

func TestProfileBatchIngestorRejectsSamplesWithoutTimeRange(t *testing.T) {
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	result, err := ingestor.Ingest(context.Background(), ProfileBatchRequest{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		Samples: []clickhouse.ProfileSample{{
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StackID:     "stack-1",
			Value:       10,
		}},
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected missing time range to be rejected, got %+v", result)
	}
}

func TestProfileBatchIngestorDoesNotRecordUnattributableRejection(t *testing.T) {
	ingestion := clickhouse.NewIngestionRepository()
	ingestor := ProfileBatchIngestor{
		Profiles:  clickhouse.NewProfileRepository(),
		Ingestion: ingestion,
	}
	result, err := ingestor.Ingest(context.Background(), ProfileBatchRequest{})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected rejected, got %+v", result)
	}
	batches, err := ingestion.ListIngestionBatches(context.Background(), clickhouse.IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) != 0 {
		t.Fatalf("unattributable rejection should not be recorded, got %+v", batches)
	}
}

type failingProfileStore struct {
	err    error
	writes int
}

func (s *failingProfileStore) InsertProfileBatch(context.Context, string, []clickhouse.ProfileSample) error {
	s.writes++
	return s.err
}

func TestProfileBatchIngestorAllowsRetryAfterPayloadWriteFailure(t *testing.T) {
	store := &failingProfileStore{err: clickhouse.ErrRetryableStorage}
	ingestion := clickhouse.NewIngestionRepository()
	req := ProfileBatchRequest{
		BatchID:     "batch-1",
		CollectorID: "collector-a",
		Samples: []clickhouse.ProfileSample{{
			BatchID:     "batch-1",
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			ProfileType: domain.ProfileTypeCPU,
			StartedAt:   time.Unix(100, 0),
			EndedAt:     time.Unix(160, 0),
			StackID:     "stack-1",
			Value:       10,
		}},
	}
	ingestor := ProfileBatchIngestor{Profiles: store, Ingestion: ingestion}
	result, err := ingestor.Ingest(context.Background(), req)
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRetryable {
		t.Fatalf("expected retryable write failure, got %+v", result)
	}

	profiles := clickhouse.NewProfileRepository()
	ingestor.Profiles = profiles
	result, err = ingestor.Ingest(context.Background(), req)
	if err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected retry after claim to write payload, got %+v err=%v", result, err)
	}
	samples, err := profiles.QuerySamples(context.Background(), clickhouse.ProfileQuery{Namespace: "prod", Service: "checkout"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected retried payload write, got %+v", samples)
	}
	batches, err := ingestion.ListIngestionBatches(context.Background(), clickhouse.IngestionQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected one final ingestion state, got %+v", batches)
	}
	if batches[0].Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected final accepted state, got %+v", batches[0])
	}
}

func TestTargetStatusBatchIngestorDeduplicatesBeforeWrite(t *testing.T) {
	statuses := clickhouse.NewStatusRepository()
	ingestor := TargetStatusIngestor{
		Statuses:  statuses,
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	req := TargetStatusBatchRequest{
		BatchID:     "status-batch-1",
		CollectorID: "collector-a",
		Statuses: []clickhouse.TargetStatus{{
			Target:       domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-a", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			StatusAt:     time.Unix(100, 0),
			DesiredState: domain.TargetDesiredStateEnabled,
			Reason:       domain.StatusReasonAccepted,
		}},
	}
	if result, err := ingestor.Ingest(context.Background(), req); err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected accepted, got %+v err=%v", result, err)
	}
	if result, err := ingestor.Ingest(context.Background(), req); err != nil || result.Status != clickhouse.IngestionDuplicate {
		t.Fatalf("expected duplicate, got %+v err=%v", result, err)
	}
	changed := req
	changed.Statuses = []clickhouse.TargetStatus{{
		Target:       domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-b", ProcessID: 2, JVMStartTime: time.Unix(2, 0)},
		StatusAt:     time.Unix(101, 0),
		DesiredState: domain.TargetDesiredStateEnabled,
		Reason:       domain.StatusReasonAccepted,
	}}
	if result, err := ingestor.Ingest(context.Background(), changed); err != nil || result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected conflicting batch rejected, got %+v err=%v", result, err)
	}
	got, err := statuses.LatestByService(context.Background(), clickhouse.TargetStatusQuery{Namespace: "prod", Service: "checkout"})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(got) != 1 || got[0].Target.Pod != "pod-a" || got[0].BatchID != req.BatchID {
		t.Fatalf("unexpected status rows after duplicate/conflict: %+v", got)
	}
}

func TestTargetStatusBatchIngestorRejectsChildBatchConflict(t *testing.T) {
	ingestor := TargetStatusIngestor{
		Statuses:  clickhouse.NewStatusRepository(),
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	result, err := ingestor.Ingest(context.Background(), TargetStatusBatchRequest{
		BatchID:     "status-batch-1",
		CollectorID: "collector-a",
		Statuses: []clickhouse.TargetStatus{{
			BatchID:      "other-batch",
			Target:       domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-a", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			StatusAt:     time.Unix(100, 0),
			DesiredState: domain.TargetDesiredStateEnabled,
			Reason:       domain.StatusReasonAccepted,
		}},
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected child batch conflict rejected, got %+v", result)
	}
}

func TestThreadSnapshotBatchIngestorDeduplicatesBeforeWrite(t *testing.T) {
	threads := clickhouse.NewThreadRepository()
	ingestor := ThreadSnapshotIngestor{
		Threads:   threads,
		Ingestion: clickhouse.NewIngestionRepository(),
	}
	req := ThreadSnapshotBatchRequest{
		BatchID:     "thread-batch-1",
		CollectorID: "collector-a",
		Snapshots: []clickhouse.ThreadSnapshot{{
			Target:     domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-a", ProcessID: 1, JVMStartTime: time.Unix(1, 0)},
			SnapshotAt: time.Unix(100, 0),
			ThreadID:   1,
			ThreadName: "main",
			State:      "RUNNABLE",
		}},
	}
	if result, err := ingestor.Ingest(context.Background(), req); err != nil || result.Status != clickhouse.IngestionAccepted {
		t.Fatalf("expected accepted, got %+v err=%v", result, err)
	}
	if result, err := ingestor.Ingest(context.Background(), req); err != nil || result.Status != clickhouse.IngestionDuplicate {
		t.Fatalf("expected duplicate, got %+v err=%v", result, err)
	}
	changed := req
	changed.Snapshots = []clickhouse.ThreadSnapshot{{
		Target:     domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-b", ProcessID: 2, JVMStartTime: time.Unix(2, 0)},
		SnapshotAt: time.Unix(101, 0),
		ThreadID:   2,
		ThreadName: "worker",
		State:      "BLOCKED",
	}}
	if result, err := ingestor.Ingest(context.Background(), changed); err != nil || result.Status != clickhouse.IngestionRejected {
		t.Fatalf("expected conflicting batch rejected, got %+v err=%v", result, err)
	}
	got, err := threads.ListSnapshots(context.Background(), "prod", "checkout")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(got) != 1 || got[0].ThreadName != "main" || got[0].BatchID != req.BatchID {
		t.Fatalf("unexpected thread rows after duplicate/conflict: %+v", got)
	}
}
