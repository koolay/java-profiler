package clickhouse

import (
	_ "embed"
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
