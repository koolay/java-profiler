package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/koolay/java-profiler/domain"
)

type SQLRepository struct {
	db *sql.DB
}

func OpenSQLRepository(dsn string) (*SQLRepository, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	return &SQLRepository{db: db}, nil
}

func (r *SQLRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLRepository) ApplySchema(ctx context.Context) error {
	schema, err := InitialSchema()
	if err != nil {
		return err
	}
	for _, stmt := range SplitStatements(schema) {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLRepository) InsertProfileBatch(ctx context.Context, batchID string, samples []ProfileSample) error {
	seenStacks := map[string]bool{}
	for _, sample := range samples {
		stackKey := profileStackKey(sample)
		if !seenStacks[stackKey] {
			if _, err := r.db.ExecContext(ctx, `
				INSERT INTO java_profiler_profile_stacks
				(cluster, namespace, service, pod, container, node, process_id, jvm_start_time, stack_id, frames)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				sample.Target.Cluster, sample.Target.Namespace, sample.Target.Service, sample.Target.Pod, sample.Target.Container, sample.Target.Node, sample.Target.ProcessID, sample.Target.JVMStartTime, sample.StackID, sample.Frames); err != nil {
				return err
			}
			seenStacks[stackKey] = true
		}
		truncated := uint8(0)
		if sample.Truncated {
			truncated = 1
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO java_profiler_profile_samples
			(batch_id, cluster, namespace, service, pod, container, node, process_id, jvm_start_time, profile_type, started_at, ended_at, stack_id, sample_value, truncated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			batchID, sample.Target.Cluster, sample.Target.Namespace, sample.Target.Service, sample.Target.Pod, sample.Target.Container, sample.Target.Node, sample.Target.ProcessID, sample.Target.JVMStartTime, sample.ProfileType.String(), sample.StartedAt, sample.EndedAt, sample.StackID, sample.Value, truncated); err != nil {
			return err
		}
	}
	return nil
}

func profileStackKey(sample ProfileSample) string {
	return strings.Join([]string{
		sample.Target.Cluster,
		sample.Target.Namespace,
		sample.Target.Service,
		sample.Target.Pod,
		sample.Target.Container,
		sample.Target.Node,
		sample.Target.JVMStartTime.UTC().Format(time.RFC3339Nano),
		sample.StackID,
	}, "\x00") + "\x00" + strconv.FormatInt(int64(sample.Target.ProcessID), 10)
}

func (r *SQLRepository) QuerySamples(ctx context.Context, q ProfileQuery) ([]ProfileSample, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	query := `
		SELECT s.batch_id, s.cluster, s.namespace, s.service, s.pod, s.container, s.node, s.process_id, s.jvm_start_time, s.profile_type, s.started_at, s.ended_at, s.stack_id, s.sample_value, s.truncated, any(st.frames)
		FROM java_profiler_profile_samples s
		LEFT JOIN java_profiler_profile_stacks st
		  ON s.cluster = st.cluster
		 AND s.namespace = st.namespace
		 AND s.service = st.service
		 AND s.pod = st.pod
		 AND s.container = st.container
		 AND s.node = st.node
		 AND s.process_id = st.process_id
		 AND s.jvm_start_time = st.jvm_start_time
		 AND s.stack_id = st.stack_id
		WHERE (? = '' OR s.namespace = ?) AND (? = '' OR s.service = ?) AND (? = '' OR s.pod = ?) AND (? = '' OR s.profile_type = ?)
		  AND (? = 1 OR s.ended_at >= ?) AND (? = 1 OR s.started_at <= ?)
		GROUP BY s.batch_id, s.cluster, s.namespace, s.service, s.pod, s.container, s.node, s.process_id, s.jvm_start_time, s.profile_type, s.started_at, s.ended_at, s.stack_id, s.sample_value, s.truncated
		ORDER BY s.started_at DESC
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query,
		q.Namespace, q.Namespace,
		q.Service, q.Service,
		q.Pod, q.Pod,
		q.ProfileType.String(), q.ProfileType.String(),
		zeroTimeFlag(q.Start), q.Start,
		zeroTimeFlag(q.End), q.End,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProfileSample
	for rows.Next() {
		var sample ProfileSample
		var profileType string
		var truncated uint8
		if err := rows.Scan(&sample.BatchID, &sample.Target.Cluster, &sample.Target.Namespace, &sample.Target.Service, &sample.Target.Pod, &sample.Target.Container, &sample.Target.Node, &sample.Target.ProcessID, &sample.Target.JVMStartTime, &profileType, &sample.StartedAt, &sample.EndedAt, &sample.StackID, &sample.Value, &truncated, &sample.Frames); err != nil {
			return nil, err
		}
		sample.ProfileType = profileTypeFromString(profileType)
		sample.Truncated = truncated == 1
		out = append(out, sample)
	}
	return out, rows.Err()
}

func (r *SQLRepository) InsertSnapshots(ctx context.Context, snapshots []ThreadSnapshot, deadlocks []DeadlockEvent) error {
	for _, snapshot := range snapshots {
		daemon := uint8(0)
		if snapshot.Daemon {
			daemon = 1
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO java_profiler_thread_snapshots
			(batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, snapshot_at, thread_id, native_thread_id, thread_name, daemon, thread_state, stack_frames, lock_owner, blocked_lock, waited_lock, deadlock_cycle_id, cpu_time_ns, user_cpu_time_ns)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			snapshot.BatchID,
			snapshot.Target.Cluster,
			snapshot.Target.Namespace,
			snapshot.Target.Service,
			snapshot.Target.Pod,
			snapshot.Target.Container,
			snapshot.Target.ProcessID,
			snapshot.Target.JVMStartTime,
			snapshot.SnapshotAt,
			snapshot.ThreadID,
			snapshot.NativeThreadID,
			snapshot.ThreadName,
			daemon,
			snapshot.State,
			snapshot.StackFrames,
			snapshot.LockOwner,
			snapshot.BlockedLock,
			snapshot.WaitedLock,
			snapshot.DeadlockCycleID,
			snapshot.CPUTimeNS,
			snapshot.UserCPUTimeNS,
		); err != nil {
			return err
		}
	}
	for _, event := range deadlocks {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO java_profiler_deadlock_events
			(event_id, cluster, namespace, service, pod, process_id, jvm_start_time, event_at, cycle_id, involved_threads, locks, blocking_frames)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			event.EventID,
			event.Target.Cluster,
			event.Target.Namespace,
			event.Target.Service,
			event.Target.Pod,
			event.Target.ProcessID,
			event.Target.JVMStartTime,
			event.EventAt,
			event.CycleID,
			event.InvolvedThreads,
			event.Locks,
			event.BlockingFrames,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLRepository) ListSnapshots(ctx context.Context, namespace, service string) ([]ThreadSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, snapshot_at, thread_id, native_thread_id, thread_name, daemon, thread_state, stack_frames, lock_owner, blocked_lock, waited_lock, deadlock_cycle_id, cpu_time_ns, user_cpu_time_ns
		FROM java_profiler_thread_snapshots
		WHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
		ORDER BY snapshot_at DESC
		LIMIT 1000`,
		namespace, namespace,
		service, service,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ThreadSnapshot
	for rows.Next() {
		var snapshot ThreadSnapshot
		var daemon uint8
		var cpu sql.NullInt64
		var userCPU sql.NullInt64
		if err := rows.Scan(
			&snapshot.BatchID,
			&snapshot.Target.Cluster,
			&snapshot.Target.Namespace,
			&snapshot.Target.Service,
			&snapshot.Target.Pod,
			&snapshot.Target.Container,
			&snapshot.Target.ProcessID,
			&snapshot.Target.JVMStartTime,
			&snapshot.SnapshotAt,
			&snapshot.ThreadID,
			&snapshot.NativeThreadID,
			&snapshot.ThreadName,
			&daemon,
			&snapshot.State,
			&snapshot.StackFrames,
			&snapshot.LockOwner,
			&snapshot.BlockedLock,
			&snapshot.WaitedLock,
			&snapshot.DeadlockCycleID,
			&cpu,
			&userCPU,
		); err != nil {
			return nil, err
		}
		snapshot.Daemon = daemon == 1
		if cpu.Valid {
			value := uint64(cpu.Int64)
			snapshot.CPUTimeNS = &value
		}
		if userCPU.Valid {
			value := uint64(userCPU.Int64)
			snapshot.UserCPUTimeNS = &value
		}
		out = append(out, snapshot)
	}
	return out, rows.Err()
}

func (r *SQLRepository) ListDeadlocks(ctx context.Context, namespace, service string) ([]DeadlockEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT event_id, cluster, namespace, service, pod, process_id, jvm_start_time, event_at, cycle_id, involved_threads, locks, blocking_frames
		FROM java_profiler_deadlock_events
		WHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
		ORDER BY event_at DESC
		LIMIT 500`,
		namespace, namespace,
		service, service,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeadlockEvent
	for rows.Next() {
		var event DeadlockEvent
		if err := rows.Scan(
			&event.EventID,
			&event.Target.Cluster,
			&event.Target.Namespace,
			&event.Target.Service,
			&event.Target.Pod,
			&event.Target.ProcessID,
			&event.Target.JVMStartTime,
			&event.EventAt,
			&event.CycleID,
			&event.InvolvedThreads,
			&event.Locks,
			&event.BlockingFrames,
		); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *SQLRepository) InsertStatuses(ctx context.Context, statuses []TargetStatus) error {
	if len(statuses) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO java_profiler_target_status
		(batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, status_at, desired_state, reason, message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, status := range statuses {
		if _, err := stmt.ExecContext(ctx,
			status.BatchID,
			status.Target.Cluster,
			status.Target.Namespace,
			status.Target.Service,
			status.Target.Pod,
			status.Target.Container,
			status.Target.ProcessID,
			status.Target.JVMStartTime,
			status.StatusAt,
			string(status.DesiredState),
			string(status.Reason),
			status.Message,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLRepository) LatestByService(ctx context.Context, q TargetStatusQuery) ([]TargetStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, status_at, desired_state, reason, message
		FROM
		(
			SELECT *, row_number() OVER (
				PARTITION BY cluster, namespace, service, pod, container, process_id, jvm_start_time
				ORDER BY status_at DESC
			) AS rn
			FROM java_profiler_target_status
			WHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
			  AND (? = 1 OR status_at >= ?) AND (? = 1 OR status_at <= ?)
		)
		WHERE rn = 1
		ORDER BY status_at DESC
		LIMIT 500`,
		q.Namespace, q.Namespace,
		q.Service, q.Service,
		zeroTimeFlag(q.Start), q.Start,
		zeroTimeFlag(q.End), q.End,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TargetStatus
	for rows.Next() {
		var status TargetStatus
		var desiredState string
		var reason string
		if err := rows.Scan(
			&status.BatchID,
			&status.Target.Cluster,
			&status.Target.Namespace,
			&status.Target.Service,
			&status.Target.Pod,
			&status.Target.Container,
			&status.Target.ProcessID,
			&status.Target.JVMStartTime,
			&status.StatusAt,
			&desiredState,
			&reason,
			&status.Message,
		); err != nil {
			return nil, err
		}
		status.DesiredState = domain.TargetDesiredState(desiredState)
		status.Reason = domain.StatusReason(reason)
		out = append(out, status)
	}
	return out, rows.Err()
}

func zeroTimeFlag(value time.Time) uint8 {
	if value.IsZero() {
		return 1
	}
	return 0
}

func (r *SQLRepository) Record(ctx context.Context, batch IngestionBatch) (IngestionStatus, error) {
	var existingHash string
	var existingStatus string
	err := r.db.QueryRowContext(ctx, `
		SELECT payload_hash, status
		FROM java_profiler_ingestion_batches
		WHERE batch_id = ? AND batch_type = ?
		ORDER BY received_at DESC
		LIMIT 1`, batch.BatchID, batch.BatchType).Scan(&existingHash, &existingStatus)
	if err == nil {
		if existingHash == batch.PayloadHash {
			if IngestionStatus(existingStatus) == IngestionClaimed || IngestionStatus(existingStatus) == IngestionRetryable {
				if batch.Status == IngestionAccepted || batch.Status == IngestionRetryable || batch.Status == IngestionRejected {
					if err := r.insertIngestionBatch(ctx, batch); err != nil {
						return "", err
					}
					return batch.Status, nil
				}
				return IngestionClaimed, nil
			}
			return IngestionDuplicate, nil
		}
		if batch.Status == IngestionRejected {
			if err := r.insertIngestionBatch(ctx, batch); err != nil {
				return "", err
			}
			return batch.Status, nil
		}
		return IngestionRejected, nil
	}
	if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(err.Error(), "no rows") {
		return "", err
	}
	if batch.ReceivedAt.IsZero() {
		batch.ReceivedAt = time.Now().UTC()
	}
	if err := r.insertIngestionBatch(ctx, batch); err != nil {
		return "", err
	}
	return batch.Status, nil
}

func (r *SQLRepository) insertIngestionBatch(ctx context.Context, batch IngestionBatch) error {
	retryable := uint8(0)
	if batch.Retryable {
		retryable = 1
	}
	truncated := uint8(0)
	if batch.Truncated {
		truncated = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO java_profiler_ingestion_batches
		(batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		batch.BatchID, batch.CollectorID, batch.BatchType, batch.ReceivedAt, batch.Status, retryable, batch.PayloadHash, batch.Message, batch.RawSampleCount, batch.AggregatedSampleCount, batch.BatchSampleCount, batch.DroppedSampleCount, batch.DroppedStackCount, truncated)
	return err
}

func (r *SQLRepository) ListIngestionBatches(ctx context.Context, q IngestionQuery) ([]IngestionBatch, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated
		FROM java_profiler_ingestion_batches
		ORDER BY received_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IngestionBatch
	for rows.Next() {
		var batch IngestionBatch
		var batchType string
		var status string
		var retryable uint8
		var truncated uint8
		if err := rows.Scan(&batch.BatchID, &batch.CollectorID, &batchType, &batch.ReceivedAt, &status, &retryable, &batch.PayloadHash, &batch.Message, &batch.RawSampleCount, &batch.AggregatedSampleCount, &batch.BatchSampleCount, &batch.DroppedSampleCount, &batch.DroppedStackCount, &truncated); err != nil {
			return nil, err
		}
		batch.BatchType = domain.BatchType(batchType)
		batch.Status = IngestionStatus(status)
		batch.Retryable = retryable == 1
		batch.Truncated = truncated == 1
		out = append(out, batch)
	}
	return out, rows.Err()
}

func profileTypeFromString(value string) domain.ProfileType {
	pt := domain.ProfileType(value)
	if !pt.IsValid() {
		return ""
	}
	return pt
}
