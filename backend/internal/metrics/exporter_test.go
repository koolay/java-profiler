package metrics

import (
	"strings"
	"testing"
)

func TestExporterRendersPrometheusText(t *testing.T) {
	exporter := NewExporter()
	exporter.Inc("java_profiler_ingestion_success_total")
	exporter.Set("java_profiler_clickhouse_ttl_lag_seconds", 7)
	got := exporter.Snapshot()
	if !strings.Contains(got, "java_profiler_ingestion_success_total 1") {
		t.Fatalf("missing counter: %s", got)
	}
	if !strings.Contains(got, "java_profiler_clickhouse_ttl_lag_seconds 7") {
		t.Fatalf("missing gauge: %s", got)
	}
}
