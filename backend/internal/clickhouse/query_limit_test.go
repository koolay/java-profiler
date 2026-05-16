package clickhouse

import (
	"context"
	"testing"
	"time"

	root "github.com/koolay/java-profiler/domain"
)

func TestThreadRepositoryListSnapshotsLimitedStopsAtLimit(t *testing.T) {
	repo := NewThreadRepository()
	if err := repo.InsertSnapshots(context.Background(), []ThreadSnapshot{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 1},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 2},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 3},
	}, nil); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListSnapshotsLimited(context.Background(), "prod", "checkout", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two snapshots after limit, got %d", len(got))
	}
}

func TestThreadRepositoryListDeadlocksLimitedStopsAtLimit(t *testing.T) {
	repo := NewThreadRepository()
	if err := repo.InsertSnapshots(context.Background(), nil, []DeadlockEvent{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-1"},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-2"},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-3"},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListDeadlocksLimited(context.Background(), "prod", "checkout", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two deadlocks after limit, got %d", len(got))
	}
}

func TestStatusRepositoryLatestByServiceAppliesLimit(t *testing.T) {
	repo := NewStatusRepository()
	now := time.Unix(100, 0).UTC()
	if err := repo.InsertStatuses(context.Background(), []TargetStatus{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-1"}, StatusAt: now},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-2"}, StatusAt: now.Add(time.Second)},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "pod-3"}, StatusAt: now.Add(2 * time.Second)},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.LatestByService(context.Background(), TargetStatusQuery{Namespace: "prod", Service: "checkout", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two statuses after limit, got %d", len(got))
	}
	if got[0].Target.Pod != "pod-3" {
		t.Fatalf("expected latest status first, got %s", got[0].Target.Pod)
	}
}
