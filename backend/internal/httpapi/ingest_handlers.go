package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/koolay/java-profiler/backend/internal/app"
	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
)

type IngestHandlers struct {
	Profiles        app.ProfileBatchIngestor
	ThreadSnapshots app.ThreadSnapshotIngestor
	TargetStatuses  app.TargetStatusIngestor
	Metrics         *metrics.Exporter
}

func (h IngestHandlers) ProfileBatch(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	defer r.Body.Close()
	var req app.ProfileBatchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<20)).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "profile batch too large", http.StatusRequestEntityTooLarge)
			recordMetric(h.Metrics, "java_profiler_http_ingest_profile_requests_total", 1)
			recordMetric(h.Metrics, "java_profiler_http_ingest_profile_errors_total", 1)
			recordMetric(h.Metrics, "java_profiler_http_ingest_profile_duration_seconds_total", time.Since(started).Seconds())
			return
		}
		http.Error(w, "invalid json", http.StatusBadRequest)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	result, err := h.Profiles.Ingest(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	recordMetric(h.Metrics, "java_profiler_http_ingest_profile_requests_total", 1)
	recordMetric(h.Metrics, "java_profiler_http_ingest_profile_duration_seconds_total", time.Since(started).Seconds())
	recordMetric(h.Metrics, "java_profiler_http_ingest_profile_samples_total", float64(len(req.Samples)))
	status := http.StatusAccepted
	if result.Status == clickhouse.IngestionRejected {
		status = http.StatusBadRequest
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_rejected_total", 1)
	}
	if result.Status == clickhouse.IngestionRetryable {
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_retryable_total", 1)
	}
	if result.Status == clickhouse.IngestionAccepted {
		recordMetric(h.Metrics, "java_profiler_http_ingest_profile_accepted_total", 1)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (h IngestHandlers) TargetStatusBatch(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	defer r.Body.Close()
	var req app.TargetStatusBatchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	result, err := h.TargetStatuses.Ingest(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_requests_total", 1)
	recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_duration_seconds_total", time.Since(started).Seconds())
	recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_rows_total", float64(len(req.Statuses)))
	status := http.StatusAccepted
	if result.Status == clickhouse.IngestionRejected {
		status = http.StatusBadRequest
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_rejected_total", 1)
	}
	if result.Status == clickhouse.IngestionRetryable {
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_retryable_total", 1)
	}
	if result.Status == clickhouse.IngestionAccepted {
		recordMetric(h.Metrics, "java_profiler_http_ingest_target_status_accepted_total", 1)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (h IngestHandlers) ThreadSnapshotBatch(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	defer r.Body.Close()
	var req app.ThreadSnapshotBatchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	result, err := h.ThreadSnapshots.Ingest(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_requests_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_errors_total", 1)
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_duration_seconds_total", time.Since(started).Seconds())
		return
	}
	recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_requests_total", 1)
	recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_duration_seconds_total", time.Since(started).Seconds())
	recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_rows_total", float64(len(req.Snapshots)))
	status := http.StatusAccepted
	if result.Status == clickhouse.IngestionRejected {
		status = http.StatusBadRequest
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_rejected_total", 1)
	}
	if result.Status == clickhouse.IngestionRetryable {
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_retryable_total", 1)
	}
	if result.Status == clickhouse.IngestionAccepted {
		recordMetric(h.Metrics, "java_profiler_http_ingest_thread_snapshot_accepted_total", 1)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}
