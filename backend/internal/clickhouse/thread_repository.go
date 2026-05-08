package clickhouse

import (
	"context"
	"sync"

	profiling "github.com/koolay/java-profiler/contracts/profiling"
)

type ThreadSnapshot = profiling.ThreadSnapshot
type DeadlockEvent = profiling.DeadlockEvent

type ThreadRepository struct {
	mu        sync.RWMutex
	snapshots []ThreadSnapshot
	deadlocks []DeadlockEvent
}

func NewThreadRepository() *ThreadRepository { return &ThreadRepository{} }

func (r *ThreadRepository) InsertSnapshots(_ context.Context, snapshots []ThreadSnapshot, deadlocks []DeadlockEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots = append(r.snapshots, snapshots...)
	r.deadlocks = append(r.deadlocks, deadlocks...)
	return nil
}

func (r *ThreadRepository) ListSnapshots(_ context.Context, namespace, service string) ([]ThreadSnapshot, error) {
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
	}
	return out, nil
}

func (r *ThreadRepository) ListDeadlocks(_ context.Context, namespace, service string) ([]DeadlockEvent, error) {
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
	}
	return out, nil
}
