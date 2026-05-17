package clickhouse

import (
	"context"
	"sync"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
)

type ThreadSnapshot = profiling.ThreadSnapshot
type DeadlockEvent = profiling.DeadlockEvent
type JVMEvent = profiling.JVMEvent

type ThreadRepository struct {
	mu        sync.RWMutex
	snapshots []ThreadSnapshot
	deadlocks []DeadlockEvent
	jvmEvents []JVMEvent
}

func (r *ThreadRepository) InsertJVMEvents(_ context.Context, events []JVMEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jvmEvents = append(r.jvmEvents, events...)
	return nil
}

func (r *ThreadRepository) QueryJVMEvents(_ context.Context, q JVMEventQuery) ([]JVMEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	out := make([]JVMEvent, 0)
	for _, event := range r.jvmEvents {
		if !jvmEventMatches(event, q) {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func NewThreadRepository() *ThreadRepository { return &ThreadRepository{} }

func (r *ThreadRepository) InsertSnapshots(_ context.Context, snapshots []ThreadSnapshot, deadlocks []DeadlockEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, snapshots...)
	r.deadlocks = append(r.deadlocks, deadlocks...)
	return nil
}

func (r *ThreadRepository) ListSnapshots(ctx context.Context, namespace, service string) ([]ThreadSnapshot, error) {
	return r.ListSnapshotsLimited(ctx, namespace, service, 0)
}

func (r *ThreadRepository) ListSnapshotsLimited(_ context.Context, namespace, service string, limit int) ([]ThreadSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ThreadSnapshot, 0)
	for _, snapshot := range r.snapshots {
		if namespace != "" && snapshot.Target.Namespace != namespace {
			continue
		}
		if service != "" && snapshot.Target.Service != service {
			continue
		}
		out = append(out, snapshot)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *ThreadRepository) ListDeadlocks(ctx context.Context, namespace, service string) ([]DeadlockEvent, error) {
	return r.ListDeadlocksLimited(ctx, namespace, service, 0)
}

func (r *ThreadRepository) ListDeadlocksLimited(_ context.Context, namespace, service string, limit int) ([]DeadlockEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DeadlockEvent, 0)
	for _, event := range r.deadlocks {
		if namespace != "" && event.Target.Namespace != namespace {
			continue
		}
		if service != "" && event.Target.Service != service {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
