package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func TestCollectorUploadRequiresAuth(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "secret"}}, metrics.NewExporter())
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

func TestCollectorTargetStatusUploadAuthenticated(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "secret", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	body := `{"BatchID":"status-batch-1","CollectorID":"collector-a","Statuses":[{"batch_id":"status-batch-1","target":{"cluster":"local","namespace":"prod","workload":"checkout","pod":"checkout-1","container":"app","node":"node-a","pod_uid":"pod-1","process_id":123,"jvm_start_time":"` + now + `","runtime_vendor":"hotspot-compatible","runtime_version":"17","service":"checkout"},"status_at":"` + now + `","desired_state":"temporary","reason":"accepted","message":"HotSpot-compatible JVM discovered"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/collector/v1/target-status-batches", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted, got %d body=%s", rec.Code, rec.Body.String())
	}

	query := httptest.NewRequest(http.MethodGet, "/api/ui/v1/target-status?namespace=prod&service=checkout", nil)
	query.AddCookie(&http.Cookie{Name: "java_profiler_ui_token", Value: "ui"})
	queryRec := httptest.NewRecorder()
	server.ServeHTTP(queryRec, query)
	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected target status query to pass, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}
	if !strings.Contains(queryRec.Body.String(), "accepted") {
		t.Fatalf("expected accepted status in response, got %s", queryRec.Body.String())
	}

	filtered := httptest.NewRequest(http.MethodGet, "/api/ui/v1/target-status?namespace=prod&service=checkout&start=2099-01-01T00:00:00Z", nil)
	filtered.AddCookie(&http.Cookie{Name: "java_profiler_ui_token", Value: "ui"})
	filteredRec := httptest.NewRecorder()
	server.ServeHTTP(filteredRec, filtered)
	if filteredRec.Code != http.StatusOK {
		t.Fatalf("expected time-filtered target status query to pass, got %d body=%s", filteredRec.Code, filteredRec.Body.String())
	}
	if strings.Contains(filteredRec.Body.String(), "accepted") {
		t.Fatalf("expected future time range to exclude accepted status, got %s", filteredRec.Body.String())
	}
}

func TestCollectorUploadAuthenticated(t *testing.T) {
	exporter := metrics.NewExporter()
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "secret"}}, exporter)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"batch_id":"batch-1","collector_id":"collector-a","samples":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/collector/v1/profile-batches", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected accepted, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(exporter.Snapshot(), "java_profiler_http_ingest_profile_requests_total 1") {
		t.Fatalf("expected ingest request metric in snapshot: %s", exporter.Snapshot())
	}
}

func TestCollectorProfileUploadTooLarge(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "secret"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	body := io.MultiReader(
		strings.NewReader(`{"batch_id":"batch-oversized","collector_id":"collector-a","samples":[],"padding":"`),
		&repeatingByteReader{remaining: (64 << 20) + 1, value: 'x'},
		strings.NewReader(`"}`),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/collector/v1/profile-batches", body)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected request entity too large, got %d body=%s", rec.Code, rec.Body.String())
	}
}

type repeatingByteReader struct {
	remaining int
	value     byte
}

func (r *repeatingByteReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = r.value
	}
	r.remaining -= len(p)
	return len(p), nil
}
