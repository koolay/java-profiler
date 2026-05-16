package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/koolay/java-profiler/backend/internal/app"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
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

func TestTopStacksRouteReturnsSelfAndTotalRows(t *testing.T) {
	exporter := metrics.NewExporter()
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, exporter)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(100, 0).UTC()
	payload := app.ProfileBatchRequest{
		BatchID:     "batch-top-stacks",
		CollectorID: "collector-a",
		ReceivedAt:  now,
		Metadata: profiling.ProfileBatchMetadata{
			WindowRawSampleCount:        2,
			WindowAggregatedSampleCount: 2,
			BatchSampleCount:            2,
		},
		Samples: []profiling.ProfileSample{
			{
				Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1"},
				ProfileType: domain.ProfileTypeCPU,
				StartedAt:   now,
				EndedAt:     now.Add(time.Second),
				StackID:     "a",
				Frames:      []string{"root", "Demo.handleWork:93", "Demo.burnCpu:188"},
				Value:       8,
			},
			{
				Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1"},
				ProfileType: domain.ProfileTypeCPU,
				StartedAt:   now,
				EndedAt:     now.Add(time.Second),
				StackID:     "b",
				Frames:      []string{"root", "Demo.handleWork:93", "Demo.writeJson:232"},
				Value:       2,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	ingestReq := httptest.NewRequest(http.MethodPost, "/api/collector/v1/profile-batches", bytes.NewReader(body))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestReq.Header.Set("Authorization", "Bearer collector")
	ingestRec := httptest.NewRecorder()
	server.ServeHTTP(ingestRec, ingestReq)
	if ingestRec.Code != http.StatusAccepted {
		t.Fatalf("expected profile ingest to pass, got %d body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/top-stacks?namespace=prod&service=checkout&pod=checkout-1&profile_type=java_cpu_nanoseconds", nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected top stacks query to pass, got %d body=%s", rec.Code, rec.Body.String())
	}

	var rows []struct {
		Symbol       string `json:"symbol"`
		Self         uint64 `json:"self"`
		Total        uint64 `json:"total"`
		SelfPercent  string `json:"self_percent"`
		TotalPercent string `json:"total_percent"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 || rows[0].Symbol != "Demo.handleWork" {
		t.Fatalf("expected handleWork to rank first, got %#v", rows)
	}
	if rows[0].Self != 0 || rows[0].Total != 10 || rows[0].TotalPercent != "100.0%" {
		t.Fatalf("unexpected handleWork row: %#v", rows[0])
	}
	snapshot := exporter.Snapshot()
	if !strings.Contains(snapshot, "java_profiler_http_query_top_stacks_requests_total 1") {
		t.Fatalf("missing top stacks request metric: %s", snapshot)
	}
	if !strings.Contains(snapshot, "java_profiler_query_top_stacks_rows_total 3") {
		t.Fatalf("missing top stacks row metric: %s", snapshot)
	}
}
