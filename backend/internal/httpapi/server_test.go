package httpapi

import (
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestNewServerRequiresClickHouseUnlessMemoryAllowed(t *testing.T) {
	_, err := NewServer(ServerConfig{}, metrics.NewExporter())
	if err == nil {
		t.Fatalf("expected clickhouse configuration to be required")
	}
}
