package app

import (
	"context"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	root "github.com/koolay/java-profiler/domain"
)

func TestQueryTargetStatusAppliesLimit(t *testing.T) {
	repo := clickhouse.NewStatusRepository()
	now := time.Unix(100, 0).UTC()
	if err := repo.InsertStatuses(context.Background(), []clickhouse.TargetStatus{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-1"}, StatusAt: now},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-2"}, StatusAt: now.Add(time.Second)},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-3"}, StatusAt: now.Add(2 * time.Second)},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := QueryTargetStatus(context.Background(), repo, "prod", "checkout", time.Time{}, time.Time{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two statuses after limit, got %d", len(got))
	}
}
