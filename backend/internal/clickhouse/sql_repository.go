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
	for _, stmt := range SchemaUpgradeStatements() {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := r.migrateIngestionBatchesEventTable(ctx); err != nil {
		return err
	}
	return nil
}

func (r *SQLRepository) migrateIngestionBatchesEventTable(ctx context.Context) error {
	var engine string
	err := r.db.QueryRowContext(ctx, `
		SELECT engine
		FROM system.tables
		WHERE database = currentDatabase() AND name = 'java_profiler_ingestion_batches'
		LIMIT 1`).Scan(&engine)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil
		}
		return err
	}
	if engine != "ReplacingMergeTree" {
		return nil
	}
	backupName := "java_profiler_ingestion_batches_replacing_backup_" + time.Now().UTC().Format("20060102150405")
	statements := []string{
		"DROP TABLE IF EXISTS java_profiler_ingestion_batches_v2",
		IngestionBatchesCreateTableStatement("java_profiler_ingestion_batches_v2"),
		`INSERT INTO java_profiler_ingestion_batches_v2
			(batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at, created_at, expires_at)
			SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at, created_at, expires_at
			FROM java_profiler_ingestion_batches`,
		"RENAME TABLE java_profiler_ingestion_batches TO " + backupName + ", java_profiler_ingestion_batches_v2 TO java_profiler_ingestion_batches",
	}
	for _, stmt := range statements {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLRepository) InsertProfileBatch(ctx context.Context, batchID string, samples []ProfileSample) error {
	if len(samples) == 0 {
		return nil
	}
	seenStacks := make(map[string]struct{}, len(samples))
	stackRows := make([][]any, 0, len(samples))
	sampleRows := make([][]any, 0, len(samples))
	for _, sample := range samples {
		stackKey := profileStackKey(sample)
		if _, seen := seenStacks[stackKey]; !seen {
			seenStacks[stackKey] = struct{}{}
			stackRows = append(stackRows, []any{
				sample.Target.Cluster,
				sample.Target.Namespace,
				sample.Target.Service,
				sample.Target.Pod,
				sample.Target.Container,
				sample.Target.Node,
				sample.Target.ProcessID,
				sample.Target.JVMStartTime,
				sample.StackID,
				sample.Frames,
			})
		}
		truncated := uint8(0)
		if sample.Truncated {
			truncated = 1
		}
		sampleRows = append(sampleRows, []any{
			batchID,
			sample.Target.Cluster,
			sample.Target.Namespace,
			sample.Target.Service,
			sample.Target.Pod,
			sample.Target.Container,
			sample.Target.Node,
			sample.Target.ProcessID,
			sample.Target.JVMStartTime,
			sample.ProfileType.String(),
			sample.StartedAt,
			sample.EndedAt,
			sample.StackID,
			sample.Value,
			truncated,
		})
	}
	if err := execMultiRowInsert(ctx, r.db, `
		INSERT INTO java_profiler_profile_stacks
		(cluster, namespace, service, pod, container, node, process_id, jvm_start_time, stack_id, frames)`, stackRows); err != nil {
		return err
	}
	return execMultiRowInsert(ctx, r.db, `
		INSERT INTO java_profiler_profile_samples
		(batch_id, cluster, namespace, service, pod, container, node, process_id, jvm_start_time, profile_type, started_at, ended_at, stack_id, sample_value, truncated)`, sampleRows)
}

type multiRowExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func execMultiRowInsert(ctx context.Context, exec multiRowExecutor, prefix string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := buildMultiRowInsert(prefix, rows)
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func buildMultiRowInsert(prefix string, rows [][]any) (string, []any) {
	columnCount := len(rows[0])
	var builder strings.Builder
	builder.Grow(len(prefix) + len(rows)*(columnCount*3+4))
	builder.WriteString(prefix)
	builder.WriteString(" VALUES ")
	args := make([]any, 0, len(rows)*columnCount)
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('(')
		for columnIndex, value := range row {
			if columnIndex > 0 {
				builder.WriteString(", ")
			}
			builder.WriteByte('?')
			args = append(args, value)
		}
		builder.WriteByte(')')
	}
	return builder.String(), args
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
		SELECT s.batch_id, s.cluster, s.namespace, s.service, s.pod, s.container, s.node, s.process_id, s.jvm_start_time, s.profile_type, s.started_at, s.ended_at, s.stack_id, s.sample_value, s.truncated, st.frames
		FROM java_profiler_profile_samples s
		ANY LEFT JOIN java_profiler_profile_stacks st
		  ON s.cluster = st.cluster
		 AND s.namespace = st.namespace
		 AND s.service = st.service
		 AND s.pod = st.pod
		 AND s.container = st.container
		 AND s.node = st.node
		 AND s.process_id = st.process_id
		 AND s.jvm_start_time = st.jvm_start_time
		 AND s.stack_id = st.stack_id
		PREWHERE (? = '' OR s.namespace = ?) AND (? = '' OR s.service = ?) AND (? = '' OR s.pod = ?) AND (? = '' OR s.profile_type = ?)
		  AND (? = 1 OR s.ended_at >= ?) AND (? = 1 OR s.started_at <= ?)
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

func (r *SQLRepository) QueryFlamegraphSamples(ctx context.Context, q ProfileQuery) ([]FlamegraphSample, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	query := `
		SELECT s.sample_value, st.frames
		FROM java_profiler_profile_samples s
		ANY LEFT JOIN java_profiler_profile_stacks st
		  ON s.cluster = st.cluster
		 AND s.namespace = st.namespace
		 AND s.service = st.service
		 AND s.pod = st.pod
		 AND s.container = st.container
		 AND s.node = st.node
		 AND s.process_id = st.process_id
		 AND s.jvm_start_time = st.jvm_start_time
		 AND s.stack_id = st.stack_id
		PREWHERE (? = '' OR s.namespace = ?) AND (? = '' OR s.service = ?) AND (? = '' OR s.pod = ?) AND (? = '' OR s.profile_type = ?)
		  AND (? = 1 OR s.ended_at >= ?) AND (? = 1 OR s.started_at <= ?)
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
	var out []FlamegraphSample
	for rows.Next() {
		var sample FlamegraphSample
		if err := rows.Scan(&sample.Value, &sample.Frames); err != nil {
			return nil, err
		}
		out = append(out, sample)
	}
	return out, rows.Err()
}

func (r *SQLRepository) QueryTopStackSamples(ctx context.Context, q ProfileQuery) ([]TopStackSample, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	query := `
		SELECT s.profile_type, s.sample_value, st.frames
		FROM java_profiler_profile_samples s
		ANY LEFT JOIN java_profiler_profile_stacks st
		  ON s.cluster = st.cluster
		 AND s.namespace = st.namespace
		 AND s.service = st.service
		 AND s.pod = st.pod
		 AND s.container = st.container
		 AND s.node = st.node
		 AND s.process_id = st.process_id
		 AND s.jvm_start_time = st.jvm_start_time
		 AND s.stack_id = st.stack_id
		PREWHERE (? = '' OR s.namespace = ?) AND (? = '' OR s.service = ?) AND (? = '' OR s.pod = ?) AND (? = '' OR s.profile_type = ?)
		  AND (? = 1 OR s.ended_at >= ?) AND (? = 1 OR s.started_at <= ?)
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
	var out []TopStackSample
	for rows.Next() {
		var sample TopStackSample
		var profileType string
		if err := rows.Scan(&profileType, &sample.Value, &sample.Frames); err != nil {
			return nil, err
		}
		sample.ProfileType = profileTypeFromString(profileType)
		out = append(out, sample)
	}
	return out, rows.Err()
}

func (r *SQLRepository) QueryProfileTargetSummary(ctx context.Context, q ProfileQuery) ([]ProfileTargetSummary, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 5000
	}
	query := `
		SELECT namespace, service, pod, container, process_id, jvm_start_time, profile_type, sum(sample_value) AS total_value, count() AS sample_count
		FROM java_profiler_profile_samples
		PREWHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?) AND (? = '' OR pod = ?) AND (? = '' OR profile_type = ?)
		  AND (? = 1 OR ended_at >= ?) AND (? = 1 OR started_at <= ?)
		GROUP BY namespace, service, pod, container, process_id, jvm_start_time, profile_type
		ORDER BY total_value DESC
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
	var out []ProfileTargetSummary
	var grandTotal uint64
	window := domain.TimeWindow{StartedAt: q.Start, EndsAt: q.End}
	for rows.Next() {
		var summary ProfileTargetSummary
		var profileType string
		if err := rows.Scan(&summary.Namespace, &summary.Service, &summary.Pod, &summary.Container, &summary.ProcessID, &summary.JVMStartTime, &profileType, &summary.TotalValue, &summary.SampleCount); err != nil {
			return nil, err
		}
		summary.ProfileType = profileTypeFromString(profileType)
		summary.DisplayValue = domain.FormatProfileValue(summary.ProfileType, summary.TotalValue, window)
		summary.WindowSemantics = summary.ProfileType.Semantics(window)
		grandTotal += summary.TotalValue
		out = append(out, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].PercentOfTotal = percentOfTotal(out[i].TotalValue, grandTotal)
	}
	return out, nil
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

func (r *SQLRepository) InsertJVMEvents(ctx context.Context, events []JVMEvent) error {
	if len(events) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(events))
	for _, event := range events {
		rows = append(rows, []any{
			event.EventID,
			event.BatchID,
			event.Target.Cluster,
			event.Target.Namespace,
			event.Target.Service,
			event.Target.Pod,
			event.Target.Container,
			event.Target.ProcessID,
			event.Target.JVMStartTime,
			event.EventType,
			event.EventAt,
			event.DurationNS,
			event.Collector,
			event.Action,
			event.Cause,
			event.Message,
			event.StackFrames,
		})
	}
	return execMultiRowInsert(ctx, r.db, `
		INSERT INTO java_profiler_jvm_events
		(event_id, batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, event_type, event_at, duration_ns, collector, action, cause, message, stack_frames)`, rows)
}

func (r *SQLRepository) QueryJVMEvents(ctx context.Context, q JVMEventQuery) ([]JVMEvent, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT event_id, batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, event_type, event_at, duration_ns, collector, action, cause, message, stack_frames
		FROM java_profiler_jvm_events
		PREWHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?) AND (? = '' OR pod = ?) AND (? = '' OR event_type = ?)
		  AND (? = 1 OR event_at >= ?) AND (? = 1 OR event_at <= ?)
		ORDER BY event_at DESC
		LIMIT ?`,
		q.Namespace, q.Namespace,
		q.Service, q.Service,
		q.Pod, q.Pod,
		q.EventType, q.EventType,
		zeroTimeFlag(q.Start), q.Start,
		zeroTimeFlag(q.End), q.End,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JVMEvent
	for rows.Next() {
		var event JVMEvent
		if err := rows.Scan(
			&event.EventID,
			&event.BatchID,
			&event.Target.Cluster,
			&event.Target.Namespace,
			&event.Target.Service,
			&event.Target.Pod,
			&event.Target.Container,
			&event.Target.ProcessID,
			&event.Target.JVMStartTime,
			&event.EventType,
			&event.EventAt,
			&event.DurationNS,
			&event.Collector,
			&event.Action,
			&event.Cause,
			&event.Message,
			&event.StackFrames,
		); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

const (
	defaultThreadSnapshotQueryLimit = 1000
	defaultDeadlockQueryLimit       = 500
	defaultTargetStatusQueryLimit   = 500
)

func (r *SQLRepository) ListSnapshots(ctx context.Context, namespace, service string) ([]ThreadSnapshot, error) {
	return r.ListSnapshotsLimited(ctx, namespace, service, defaultThreadSnapshotQueryLimit)
}

func (r *SQLRepository) ListSnapshotsLimited(ctx context.Context, namespace, service string, limit int) ([]ThreadSnapshot, error) {
	if limit <= 0 {
		limit = defaultThreadSnapshotQueryLimit
	}
	rows, err := r.db.QueryContext(ctx, `
			SELECT batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, snapshot_at, thread_id, native_thread_id, thread_name, daemon, thread_state, stack_frames, lock_owner, blocked_lock, waited_lock, deadlock_cycle_id, cpu_time_ns, user_cpu_time_ns
			FROM java_profiler_thread_snapshots
			PREWHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
			ORDER BY snapshot_at DESC
			LIMIT ?`,
		namespace, namespace,
		service, service,
		limit,
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
	return r.ListDeadlocksLimited(ctx, namespace, service, defaultDeadlockQueryLimit)
}

func (r *SQLRepository) ListDeadlocksLimited(ctx context.Context, namespace, service string, limit int) ([]DeadlockEvent, error) {
	if limit <= 0 {
		limit = defaultDeadlockQueryLimit
	}
	rows, err := r.db.QueryContext(ctx, `
			SELECT event_id, cluster, namespace, service, pod, process_id, jvm_start_time, event_at, cycle_id, involved_threads, locks, blocking_frames
			FROM java_profiler_deadlock_events
			PREWHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
			ORDER BY event_at DESC
			LIMIT ?`,
		namespace, namespace,
		service, service,
		limit,
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
	limit := q.Limit
	if limit <= 0 {
		limit = defaultTargetStatusQueryLimit
	}
	rows, err := r.db.QueryContext(ctx, `
			SELECT batch_id, cluster, namespace, service, pod, container, process_id, jvm_start_time, status_at, desired_state, reason, message
			FROM
		(
			SELECT *, row_number() OVER (
				PARTITION BY cluster, namespace, service, pod, container, process_id, jvm_start_time
				ORDER BY status_at DESC
				) AS rn
				FROM java_profiler_target_status
				PREWHERE (? = '' OR namespace = ?) AND (? = '' OR service = ?)
				  AND (? = 1 OR status_at >= ?) AND (? = 1 OR status_at <= ?)
				)
				WHERE rn = 1
				ORDER BY status_at DESC
				LIMIT ?`,
		q.Namespace, q.Namespace,
		q.Service, q.Service,
		zeroTimeFlag(q.Start), q.Start,
		zeroTimeFlag(q.End), q.End,
		limit,
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
		ORDER BY status_version DESC, recorded_at DESC, received_at DESC
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
	prepareIngestionBatch(&batch)
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
		(batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		batch.BatchID, batch.CollectorID, batch.BatchType, batch.ReceivedAt, batch.Status, retryable, batch.PayloadHash, batch.Message, batch.RawSampleCount, batch.AggregatedSampleCount, batch.BatchSampleCount, batch.DroppedSampleCount, batch.DroppedStackCount, truncated, batch.StatusVersion, batch.RecordedAt)
	return err
}

func (r *SQLRepository) ListIngestionBatches(ctx context.Context, q IngestionQuery) ([]IngestionBatch, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at
		FROM
		(
			SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at
			FROM java_profiler_ingestion_batches
			ORDER BY batch_id, batch_type, status_version DESC, recorded_at DESC, received_at DESC
			LIMIT 1 BY batch_id, batch_type
		)
		ORDER BY recorded_at DESC, received_at DESC
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
		if err := rows.Scan(&batch.BatchID, &batch.CollectorID, &batchType, &batch.ReceivedAt, &status, &retryable, &batch.PayloadHash, &batch.Message, &batch.RawSampleCount, &batch.AggregatedSampleCount, &batch.BatchSampleCount, &batch.DroppedSampleCount, &batch.DroppedStackCount, &truncated, &batch.StatusVersion, &batch.RecordedAt); err != nil {
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

func (r *SQLRepository) QueryIngestionHealth(ctx context.Context, q IngestionQuery) (IngestionHealthReport, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH latest_batches AS
		(
			SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at
			FROM java_profiler_ingestion_batches
			ORDER BY batch_id, batch_type, status_version DESC, recorded_at DESC, received_at DESC
			LIMIT 1 BY batch_id, batch_type
		),
		limited_batches AS
		(
			SELECT batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message, raw_sample_count, aggregated_sample_count, batch_sample_count, dropped_sample_count, dropped_stack_count, truncated, status_version, recorded_at
			FROM latest_batches
			ORDER BY recorded_at DESC, received_at DESC
			LIMIT ?
		),
		grouped_batches AS
		(
			SELECT batch_type, status, any(retryable) AS retryable, count() AS count, max(recorded_at) AS latest_at, argMax(message, recorded_at) AS last_message
			FROM limited_batches
			GROUP BY batch_type, status
		),
		totals AS
		(
			SELECT
				countIf(status = 'accepted') AS accepted,
				countIf(status = 'duplicate') AS duplicate,
				countIf(status = 'retryable') AS retryable,
				countIf(status = 'rejected') AS rejected,
				sum(dropped_sample_count) AS dropped_samples,
				sum(dropped_stack_count) AS dropped_stacks,
				countIf(truncated = 1) AS truncated_batches
			FROM limited_batches
		)
		SELECT grouped_batches.batch_type, grouped_batches.status, grouped_batches.retryable, grouped_batches.count, grouped_batches.latest_at, grouped_batches.last_message,
			totals.accepted, totals.duplicate, totals.retryable, totals.rejected, totals.dropped_samples, totals.dropped_stacks, totals.truncated_batches
		FROM grouped_batches
		CROSS JOIN totals
		ORDER BY grouped_batches.latest_at DESC, grouped_batches.batch_type, grouped_batches.status`,
		limit)
	if err != nil {
		return IngestionHealthReport{}, err
	}
	defer rows.Close()
	report := IngestionHealthReport{}
	for rows.Next() {
		var batch IngestionHealthBatch
		var batchType string
		var status string
		var retryable uint8
		if err := rows.Scan(
			&batchType,
			&status,
			&retryable,
			&batch.Count,
			&batch.LatestAt,
			&batch.LastMessage,
			&report.Totals.Accepted,
			&report.Totals.Duplicate,
			&report.Totals.Retryable,
			&report.Totals.Rejected,
			&report.Totals.DroppedSamples,
			&report.Totals.DroppedStacks,
			&report.Totals.TruncatedBatches,
		); err != nil {
			return IngestionHealthReport{}, err
		}
		batch.BatchType = domain.BatchType(batchType)
		batch.Status = IngestionStatus(status)
		batch.Retryable = retryable == 1
		report.Batches = append(report.Batches, batch)
	}
	if err := rows.Err(); err != nil {
		return IngestionHealthReport{}, err
	}
	return report, nil
}

func profileTypeFromString(value string) domain.ProfileType {
	pt := domain.ProfileType(value)
	if !pt.IsValid() {
		return ""
	}
	return pt
}
