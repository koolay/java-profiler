package app

import (
	"context"
	"testing"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	root "github.com/koolay/java-profiler/domain"
)

func TestQueryThreadDiagnosisAppliesLimitAndPartial(t *testing.T) {
	repo := clickhouse.NewThreadRepository()
	cpu := uint64(100)
	if err := repo.InsertSnapshots(context.Background(), []clickhouse.ThreadSnapshot{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 1, ThreadName: "worker-1", State: "RUNNABLE", CPUTimeNS: &cpu},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 2, ThreadName: "worker-2", State: "BLOCKED", BlockedLock: "monitor"},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, ThreadID: 3, ThreadName: "worker-3", State: "RUNNABLE", CPUTimeNS: &cpu},
	}, nil); err != nil {
		t.Fatal(err)
	}

	got, err := QueryThreadDiagnosis(context.Background(), repo, "prod", "checkout", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Partial {
		t.Fatalf("expected partial result when snapshots exceed limit: %#v", got)
	}
	if len(got.BusyThreads)+len(got.SlowThreads) != 2 {
		t.Fatalf("expected diagnosis to use two limited snapshots, got busy=%d slow=%d", len(got.BusyThreads), len(got.SlowThreads))
	}
}

func TestQueryDeadlocksAppliesLimit(t *testing.T) {
	repo := clickhouse.NewThreadRepository()
	if err := repo.InsertSnapshots(context.Background(), nil, []clickhouse.DeadlockEvent{
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-1"},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-2"},
		{Target: root.TargetIdentity{Namespace: "prod", Service: "checkout"}, EventID: "deadlock-3"},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := QueryDeadlocks(context.Background(), repo, "prod", "checkout", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two deadlocks after limit, got %d", len(got))
	}
}
