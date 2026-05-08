package clickhouse

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func InitialSchema() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "db", "clickhouse", "001_initial_profile_schema.sql")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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
