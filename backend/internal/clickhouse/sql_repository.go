package clickhouse

import (
	"context"
	"database/sql"
	"errors"
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stackStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO java_profiler_profile_stacks
		(cluster, namespace, service, pod, container, node, process_id, jvm_start_time, stack_id, frames)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stackStmt.Close()
	sampleStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO java_profiler_profile_samples
		(batch_id, cluster, namespace, service, pod, container, node, process_id, jvm_start_time, profile_type, started_at, ended_at, stack_id, sample_value, truncated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer sampleStmt.Close()
	seenStacks := map[string]bool{}
	for _, sample := range samples {
		if !seenStacks[sample.StackID] {
			if _, err := stackStmt.ExecContext(ctx, sample.Target.Cluster, sample.Target.Namespace, sample.Target.Service, sample.Target.Pod, sample.Target.Container, sample.Target.Node, sample.Target.ProcessID, sample.Target.JVMStartTime, sample.StackID, sample.Frames); err != nil {
				return err
			}
			seenStacks[sample.StackID] = true
		}
		truncated := uint8(0)
		if sample.Truncated {
			truncated = 1
		}
		if _, err := sampleStmt.ExecContext(ctx, batchID, sample.Target.Cluster, sample.Target.Namespace, sample.Target.Service, sample.Target.Pod, sample.Target.Container, sample.Target.Node, sample.Target.ProcessID, sample.Target.JVMStartTime, sample.ProfileType.String(), sample.StartedAt, sample.EndedAt, sample.StackID, sample.Value, truncated); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLRepository) QuerySamples(ctx context.Context, q ProfileQuery) ([]ProfileSample, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	query := `
		SELECT s.batch_id, s.cluster, s.namespace, s.service, s.pod, s.container, s.node, s.process_id, s.jvm_start_time, s.profile_type, s.started_at, s.ended_at, s.stack_id, s.sample_value, s.truncated, any(st.frames)
		FROM java_profiler_profile_samples s
		LEFT JOIN java_profiler_profile_stacks st ON s.stack_id = st.stack_id
		WHERE (? = '' OR s.namespace = ?) AND (? = '' OR s.service = ?) AND (? = '' OR s.pod = ?) AND (? = '' OR s.profile_type = ?)
		GROUP BY s.batch_id, s.cluster, s.namespace, s.service, s.pod, s.container, s.node, s.process_id, s.jvm_start_time, s.profile_type, s.started_at, s.ended_at, s.stack_id, s.sample_value, s.truncated
		ORDER BY s.started_at DESC
		LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, q.Namespace, q.Namespace, q.Service, q.Service, q.Pod, q.Pod, q.ProfileType.String(), q.ProfileType.String(), limit)
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

func (r *SQLRepository) Record(ctx context.Context, batch IngestionBatch) (IngestionStatus, error) {
	var existingHash string
	err := r.db.QueryRowContext(ctx, `SELECT payload_hash FROM java_profiler_ingestion_batches WHERE batch_id = ? LIMIT 1`, batch.BatchID).Scan(&existingHash)
	if err == nil {
		if existingHash == batch.PayloadHash {
			return IngestionDuplicate, nil
		}
		return IngestionRejected, nil
	}
	if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(err.Error(), "no rows") {
		return "", err
	}
	if batch.ReceivedAt.IsZero() {
		batch.ReceivedAt = time.Now().UTC()
	}
	retryable := uint8(0)
	if batch.Retryable {
		retryable = 1
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO java_profiler_ingestion_batches
		(batch_id, collector_id, batch_type, received_at, status, retryable, payload_hash, message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		batch.BatchID, batch.CollectorID, batch.BatchType, batch.ReceivedAt, batch.Status, retryable, batch.PayloadHash, batch.Message)
	if err != nil {
		return "", err
	}
	return batch.Status, nil
}

func profileTypeFromString(value string) domain.ProfileType {
	pt := domain.ProfileType(value)
	if !pt.IsValid() {
		return ""
	}
	return pt
}
