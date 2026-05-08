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
