package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestQueryRequiresAuth(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
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

func TestQueryAllowsUITokenCookie(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/flamegraph?namespace=prod&service=checkout", nil)
	req.AddCookie(&http.Cookie{Name: "java_profiler_ui_token", Value: "ui"})
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected cookie auth to pass, got %d body=%s", rec.Code, rec.Body.String())
	}
}
