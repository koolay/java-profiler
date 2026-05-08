package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestQueryRequiresAuth(t *testing.T) {
	server := NewServer(ServerConfig{Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/flamegraph?namespace=prod&service=checkout", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
}
