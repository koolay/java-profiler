package clickhouse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

func TestProfileRepositoryIdempotencyAndQuery(t *testing.T) {
	repo := NewProfileRepository()
	target := domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 42, JVMStartTime: time.Unix(10, 0)}
	sample := ProfileSample{
		BatchID:     "batch-1",
		Target:      target,
		ProfileType: domain.ProfileTypeCPU,
		StartedAt:   time.Unix(100, 0),
		EndedAt:     time.Unix(160, 0),
		StackID:     "stack-1",
		Frames:      []string{"Checkout.handle", "Repo.query"},
		Value:       100,
	}
	if err := repo.InsertProfileBatch(context.Background(), "batch-1", []ProfileSample{sample}); err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	if err := repo.InsertProfileBatch(context.Background(), "batch-1", []ProfileSample{sample}); !errors.Is(err, ErrDuplicateBatch) {
		t.Fatalf("expected duplicate batch, got %v", err)
	}
	got, err := repo.QuerySamples(context.Background(), ProfileQuery{Namespace: "prod", Service: "checkout", ProfileType: domain.ProfileTypeCPU})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(got) != 1 || got[0].StackID != "stack-1" || got[0].Value != 100 {
		t.Fatalf("unexpected query result: %+v", got)
	}
}
