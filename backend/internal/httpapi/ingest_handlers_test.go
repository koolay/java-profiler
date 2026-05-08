package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestCollectorUploadRequiresAuth(t *testing.T) {
	server, err := NewServer(ServerConfig{Auth: AuthConfig{CollectorToken: "secret"}, AllowInMemory: true}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/collector/v1/profile-batches", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
}

func TestCollectorUploadAuthenticated(t *testing.T) {
	server, err := NewServer(ServerConfig{Auth: AuthConfig{CollectorToken: "secret"}, AllowInMemory: true}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	body := `{"BatchID":"batch-1","CollectorID":"collector-a","Samples":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/collector/v1/profile-batches", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted, got %d body=%s", rec.Code, rec.Body.String())
	}
}
