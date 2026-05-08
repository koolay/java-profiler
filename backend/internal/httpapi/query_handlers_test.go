package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestQueryRequiresAuth(t *testing.T) {
	server, err := NewServer(ServerConfig{Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}, AllowInMemory: true}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/flamegraph?namespace=prod&service=checkout", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
}
