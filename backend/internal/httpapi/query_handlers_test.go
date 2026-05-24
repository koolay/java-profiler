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
		SelfDisplay  string `json:"self_display"`
		TotalDisplay string `json:"total_display"`
		SelfPercent  string `json:"self_percent"`
		TotalPercent string `json:"total_percent"`
		Semantics    struct {
			ValueUnit    string `json:"value_unit"`
			DisplayUnit  string `json:"display_unit"`
			PercentBasis string `json:"percent_basis"`
		} `json:"semantics"`
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
	if rows[0].SelfDisplay == "" || rows[0].TotalDisplay == "" || rows[0].Semantics.ValueUnit != "nanoseconds" {
		t.Fatalf("missing semantic display contract: %#v", rows[0])
	}
	snapshot := exporter.Snapshot()
	if !strings.Contains(snapshot, "java_profiler_http_query_top_stacks_requests_total 1") {
		t.Fatalf("missing top stacks request metric: %s", snapshot)
	}
	if !strings.Contains(snapshot, "java_profiler_query_top_stacks_rows_total 3") {
		t.Fatalf("missing top stacks row metric: %s", snapshot)
	}
}

func TestAllocationSummaryRouteReturnsContract(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(120, 0).UTC()
	payload := app.ProfileBatchRequest{
		BatchID:     "batch-allocation-summary",
		CollectorID: "collector-a",
		ReceivedAt:  now,
		Samples: []profiling.ProfileSample{{
			Target:      domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 1, JVMStartTime: now},
			ProfileType: domain.ProfileTypeAllocBytes,
			StartedAt:   now,
			EndedAt:     now.Add(time.Second),
			StackID:     "alloc-a",
			Frames:      []string{"root", "java/util/Arrays.copyOf:3332", "java/lang/StringBuilder.append:136"},
			Value:       4096,
		}},
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
		t.Fatalf("ingest status = %d body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/allocation-summary?namespace=prod&service=all&profile_type=java_allocation_bytes&start="+now.Add(-time.Second).Format(time.RFC3339)+"&end="+now.Add(10*time.Second).Format(time.RFC3339), nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		SchemaVersion  int `json:"schema_version"`
		RequestedScope struct {
			Service string `json:"service"`
		} `json:"requested_scope"`
		EffectiveScope struct {
			Service string `json:"service"`
		} `json:"effective_scope"`
		Coverage struct {
			HasData    bool   `json:"has_data"`
			TotalValue uint64 `json:"total_value"`
			ValueUnit  string `json:"value_unit"`
		} `json:"coverage"`
		TopPaths []struct {
			Category string `json:"category"`
		} `json:"top_paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.SchemaVersion != 1 || response.RequestedScope.Service != "all" || response.EffectiveScope.Service != "" {
		t.Fatalf("scope contract = %+v", response)
	}
	if !response.Coverage.HasData || response.Coverage.TotalValue != 4096 || response.Coverage.ValueUnit != "bytes" {
		t.Fatalf("coverage = %+v", response.Coverage)
	}
	if len(response.TopPaths) != 1 || response.TopPaths[0].Category != "string_construction" {
		t.Fatalf("top paths = %+v", response.TopPaths)
	}
}

func TestAllocationSummaryRouteRejectsInvalidProfileType(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(120, 0).UTC()
	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/allocation-summary?namespace=prod&profile_type=java_cpu_nanoseconds&start="+now.Format(time.RFC3339)+"&end="+now.Add(time.Minute).Format(time.RFC3339), nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceSummaryRouteReturnsPodJVMContributions(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0).UTC()
	payload := app.ProfileBatchRequest{
		BatchID:     "batch-summary",
		CollectorID: "collector-a",
		ReceivedAt:  now,
		Samples: []profiling.ProfileSample{
			{Target: domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 1, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"Demo.hot"}, Value: uint64(3 * time.Second)},
			{Target: domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-2", ProcessID: 2, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "b", Frames: []string{"Demo.hot"}, Value: uint64(7 * time.Second)},
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
		t.Fatalf("ingest status = %d body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/service-summary?namespace=prod&service=checkout&profile_type=java_cpu_nanoseconds&start="+now.Format(time.RFC3339)+"&end="+now.Add(10*time.Second).Format(time.RFC3339), nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Targets []struct {
			Pod            string `json:"pod"`
			DisplayValue   string `json:"display_value"`
			PercentOfTotal string `json:"percent_of_total"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Targets) != 2 || response.Targets[0].Pod != "checkout-2" || response.Targets[0].PercentOfTotal != "70.0%" {
		t.Fatalf("unexpected summary response: %+v", response)
	}
}

func TestJVMEventsRouteReturnsGCPauseEvidence(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(300, 0).UTC()
	payload := app.JVMEventBatchRequest{
		BatchID:     "batch-gc",
		CollectorID: "collector-a",
		ReceivedAt:  now,
		Events: []profiling.JVMEvent{{
			EventID:    "gc-1",
			Target:     domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 1, JVMStartTime: now},
			EventType:  "gc_pause",
			EventAt:    now.Add(time.Second),
			DurationNS: uint64(42 * time.Millisecond),
			Collector:  "G1",
			Action:     "end of minor GC",
			Cause:      "Allocation Failure",
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	ingestReq := httptest.NewRequest(http.MethodPost, "/api/collector/v1/jvm-event-batches", bytes.NewReader(body))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestReq.Header.Set("Authorization", "Bearer collector")
	ingestRec := httptest.NewRecorder()
	server.ServeHTTP(ingestRec, ingestReq)
	if ingestRec.Code != http.StatusAccepted {
		t.Fatalf("JVM event ingest status = %d body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/jvm-events?namespace=prod&service=checkout&pod=checkout-1&event_type=gc_pause", nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("JVM events status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Events []struct {
			EventID    string `json:"event_id"`
			EventType  string `json:"event_type"`
			DurationNS uint64 `json:"duration_ns"`
			Collector  string `json:"collector"`
		} `json:"events"`
		Partial bool `json:"partial"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Events) != 1 || response.Events[0].EventID != "gc-1" || response.Events[0].DurationNS != uint64(42*time.Millisecond) || response.Events[0].Collector != "G1" {
		t.Fatalf("unexpected JVM event response: %+v", response)
	}
}

func TestServiceSelectorsRouteReturnsDistinctTargets(t *testing.T) {
	server, err := NewServer(ServerConfig{AllowInMemory: true, Auth: AuthConfig{CollectorToken: "collector", UIToken: "ui"}}, metrics.NewExporter())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0).UTC()
	payload := app.ProfileBatchRequest{
		BatchID:     "batch-selectors",
		CollectorID: "collector-a",
		ReceivedAt:  now,
		Samples: []profiling.ProfileSample{
			{Target: domain.TargetIdentity{Namespace: "prod", Service: "checkout", Pod: "checkout-1", ProcessID: 1, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "a", Frames: []string{"Demo.hot"}, Value: uint64(3 * time.Second)},
			{Target: domain.TargetIdentity{Namespace: "prod", Service: "payments", Pod: "payments-1", ProcessID: 2, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "b", Frames: []string{"Demo.hot"}, Value: uint64(7 * time.Second)},
			{Target: domain.TargetIdentity{Namespace: "staging", Service: "checkout", Pod: "checkout-staging", ProcessID: 3, JVMStartTime: now}, ProfileType: domain.ProfileTypeCPU, StartedAt: now, EndedAt: now.Add(time.Second), StackID: "c", Frames: []string{"Demo.hot"}, Value: uint64(5 * time.Second)},
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
		t.Fatalf("ingest status = %d body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ui/v1/service-selectors?start="+now.Format(time.RFC3339)+"&end="+now.Add(10*time.Second).Format(time.RFC3339), nil)
	req.Header.Set("Authorization", "Bearer ui")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("selectors status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Targets []struct {
			Namespace string `json:"namespace"`
			Service   string `json:"service"`
			Pod       string `json:"pod"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Targets) != 3 {
		t.Fatalf("unexpected selector response: %+v", response)
	}
	if response.Targets[0].Namespace != "prod" || response.Targets[0].Service != "checkout" || response.Targets[0].Pod != "checkout-1" {
		t.Fatalf("unexpected first selector: %+v", response)
	}
}
