package clickhouse

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed 001_initial_profile_schema.sql
var initialSchema string

func InitialSchema() (string, error) {
	return initialSchema, nil
}

func SchemaUpgradeStatements() []string {
	return []string{
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS raw_sample_count UInt64 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS aggregated_sample_count UInt64 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS batch_sample_count UInt64 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS dropped_sample_count UInt64 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS dropped_stack_count UInt64 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS truncated UInt8 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS status_version UInt8 DEFAULT 0",
		"ALTER TABLE java_profiler_ingestion_batches ADD COLUMN IF NOT EXISTS recorded_at DateTime64(9, 'UTC') DEFAULT now64(9)",
	}
}

func IngestionBatchesCreateTableStatement(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
(
    batch_id String,
    collector_id String,
    batch_type LowCardinality(String),
    received_at DateTime64(9, 'UTC'),
    status LowCardinality(String),
    retryable UInt8,
    payload_hash String,
    message String,
    raw_sample_count UInt64 DEFAULT 0,
    aggregated_sample_count UInt64 DEFAULT 0,
    batch_sample_count UInt64 DEFAULT 0,
    dropped_sample_count UInt64 DEFAULT 0,
    dropped_stack_count UInt64 DEFAULT 0,
    truncated UInt8 DEFAULT 0,
    status_version UInt8 DEFAULT 0,
    recorded_at DateTime64(9, 'UTC') DEFAULT now64(9),
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(received_at)
ORDER BY (batch_id, batch_type, status_version, recorded_at)
TTL expires_at DELETE`, tableName)
}

func SplitStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}

func TableNames() []string {
	return []string{
		"java_profiler_profile_stacks",
		"java_profiler_profile_samples",
		"java_profiler_thread_snapshots",
		"java_profiler_deadlock_events",
		"java_profiler_target_status",
		"java_profiler_ingestion_batches",
		"java_profiler_artifact_index",
	}
}
