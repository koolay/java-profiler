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
