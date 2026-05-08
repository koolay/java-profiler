package httpapi

import (
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestNewServerRequiresExplicitStorageMode(t *testing.T) {
	_, err := NewServer(ServerConfig{Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err == nil {
		t.Fatalf("expected missing ClickHouse DSN to fail without explicit in-memory mode")
	}
}

func TestNewServerAllowsExplicitInMemoryMode(t *testing.T) {
	_, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatalf("expected explicit in-memory mode to start: %v", err)
	}
}
