package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
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
