package clickhouse

import (
	"strings"
	"testing"
	"time"
)

func TestInitialSchemaHasBoundedTTL(t *testing.T) {
	schema, err := InitialSchema()
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	for _, table := range TableNames() {
		if !strings.Contains(schema, table) {
			t.Fatalf("schema missing table %s", table)
		}
	}
	if strings.Contains(schema, "INTERVAL 8 DAY") || strings.Contains(schema, "INTERVAL 30 DAY") {
		t.Fatalf("schema must not retain collected data beyond seven days")
	}
	if !strings.Contains(schema, "INTERVAL 24 HOUR") {
		t.Fatalf("artifact index should have 24h TTL")
	}
}

func TestInitialSchemaUsesEventTableForIngestionBatches(t *testing.T) {
	schema, err := InitialSchema()
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if strings.Contains(schema, "ENGINE = ReplacingMergeTree(received_at)") {
		t.Fatalf("ingestion batches must not use ReplacingMergeTree(received_at)")
	}
	ingestionStart := strings.Index(schema, "CREATE TABLE IF NOT EXISTS java_profiler_ingestion_batches")
	if ingestionStart < 0 {
		t.Fatalf("schema missing ingestion batches table")
	}
	ingestionEnd := strings.Index(schema[ingestionStart:], "CREATE TABLE IF NOT EXISTS java_profiler_artifact_index")
	if ingestionEnd < 0 {
		t.Fatalf("schema missing artifact table after ingestion table")
	}
	ingestionSchema := schema[ingestionStart : ingestionStart+ingestionEnd]
	if !strings.Contains(ingestionSchema, "ENGINE = MergeTree") {
		t.Fatalf("ingestion batches should be a MergeTree event table:\n%s", ingestionSchema)
	}
}

func TestSchemaUpgradesAddIngestionMetadataColumns(t *testing.T) {
	upgrades := strings.Join(SchemaUpgradeStatements(), "\n")
	for _, want := range []string{
		"ADD COLUMN IF NOT EXISTS raw_sample_count",
		"ADD COLUMN IF NOT EXISTS aggregated_sample_count",
		"ADD COLUMN IF NOT EXISTS batch_sample_count",
		"ADD COLUMN IF NOT EXISTS dropped_sample_count",
		"ADD COLUMN IF NOT EXISTS dropped_stack_count",
		"ADD COLUMN IF NOT EXISTS truncated",
		"ADD COLUMN IF NOT EXISTS status_version",
		"ADD COLUMN IF NOT EXISTS recorded_at",
	} {
		if !strings.Contains(upgrades, want) {
			t.Fatalf("schema upgrades missing %q in:\n%s", want, upgrades)
		}
	}
}

func TestRetentionRepositoryReportsHealthInputs(t *testing.T) {
	status := RetentionTableStatus{
		Table:        "java_profiler_profile_samples",
		OldestRowAt:  time.Unix(100, 0),
		TTLLag:       time.Minute,
		Bytes:        1024,
		Parts:        2,
		RetentionTTL: 7 * 24 * time.Hour,
	}
	repo := NewRetentionRepository([]RetentionTableStatus{status})
	got, err := repo.Status(nil)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if len(got) != 1 || got[0].Table != status.Table || got[0].TTLLag != time.Minute {
		t.Fatalf("unexpected retention status: %+v", got)
	}
}
