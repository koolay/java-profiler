package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/koolay/java-profiler/backend/internal/app"
	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
	"github.com/koolay/java-profiler/domain"
)

type QueryHandlers struct {
	Profiles       app.ProfileQueryStore
	Threads        app.ThreadStore
	JVMEvents      app.JVMEventStore
	Statuses       app.TargetStatusQueryStore
	IngestionStore app.IngestionQueryStore
	Metrics        *metrics.Exporter
}

type queryErrorResponse struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	Field           string `json:"field,omitempty"`
	SuggestedAction string `json:"suggested_action,omitempty"`
}

func (h QueryHandlers) JVMEventsEvidence(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_jvm_events", func() (any, error) {
		return app.QueryJVMEvents(r.Context(), h.JVMEvents, clickhouse.JVMEventQuery{
			Namespace: r.URL.Query().Get("namespace"),
			Service:   r.URL.Query().Get("service"),
			Pod:       r.URL.Query().Get("pod"),
			EventType: r.URL.Query().Get("event_type"),
			Start:     parseQueryTime(r.URL.Query().Get("start")),
			End:       parseQueryTime(r.URL.Query().Get("end")),
			Limit:     parseQueryLimit(r, 500, 5000),
		})
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) Flamegraph(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_flamegraph", func() (any, error) {
		return (app.FlamegraphQuerier{Profiles: h.Profiles, Metrics: h.Metrics}).Query(r.Context(), app.FlamegraphQuery{
			Namespace:   r.URL.Query().Get("namespace"),
			Service:     r.URL.Query().Get("service"),
			Pod:         r.URL.Query().Get("pod"),
			ProfileType: domain.ProfileType(r.URL.Query().Get("profile_type")),
			Start:       parseQueryTime(r.URL.Query().Get("start")),
			End:         parseQueryTime(r.URL.Query().Get("end")),
			Limit:       1000,
			NodeLimit:   2048,
		})
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) TopStacks(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_top_stacks", func() (any, error) {
		return app.QueryTopStacks(r.Context(), h.Profiles, profileQueryFromRequest(r, 1000), h.Metrics)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) AllocationSummary(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_allocation_summary", func() (any, error) {
		return app.QueryAllocationSummary(r.Context(), h.Profiles, h.Statuses, h.IngestionStore, app.AllocationSummaryQuery{
			Namespace:      r.URL.Query().Get("namespace"),
			Service:        r.URL.Query().Get("service"),
			Pod:            r.URL.Query().Get("pod"),
			Container:      r.URL.Query().Get("container"),
			JVM:            r.URL.Query().Get("jvm"),
			ProfileType:    domain.ProfileType(r.URL.Query().Get("profile_type")),
			Start:          parseQueryTime(r.URL.Query().Get("start")),
			End:            parseQueryTime(r.URL.Query().Get("end")),
			PathLimit:      parseSpecificQueryLimit(r, "path_limit", app.DefaultAllocationPathLimit, app.MaxAllocationPathLimit),
			SelfFrameLimit: parseSpecificQueryLimit(r, "self_frame_limit", app.DefaultAllocationSelfFrameLimit, app.MaxAllocationSelfFrameLimit),
			NodeLimit:      parseSpecificQueryLimit(r, "node_limit", app.DefaultAllocationNodeLimit, app.MaxAllocationNodeLimit),
		}, h.Metrics)
	})
	if err != nil {
		var validationErr app.AllocationSummaryValidationError
		if errors.As(err, &validationErr) {
			writeQueryError(w, http.StatusBadRequest, validationErr.Code, validationErr.Message, validationErr.Field, validationErr.SuggestedAction)
			return
		}
		if errors.Is(err, app.ErrInvalidAllocationSummaryQuery) {
			writeQueryError(w, http.StatusBadRequest, "invalid_allocation_summary_query", err.Error(), "", "")
			return
		}
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) ServiceSummary(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_service_summary", func() (any, error) {
		return app.QueryServiceProfileSummary(r.Context(), h.Profiles, profileQueryFromRequest(r, 5000), h.Metrics)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) ServiceSelectors(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_service_selectors", func() (any, error) {
		return app.QueryServiceSelectors(r.Context(), h.Profiles, profileQueryFromRequest(r, 5000))
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) ThreadDiagnosis(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_thread_diagnosis", func() (any, error) {
		return app.QueryThreadDiagnosis(
			r.Context(),
			h.Threads,
			r.URL.Query().Get("namespace"),
			r.URL.Query().Get("service"),
			parseQueryLimit(r, app.DefaultThreadDiagnosisLimit, app.MaxThreadDiagnosisLimit),
		)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func profileQueryFromRequest(r *http.Request, limit int) clickhouse.ProfileQuery {
	return clickhouse.ProfileQuery{
		Namespace:   r.URL.Query().Get("namespace"),
		Service:     r.URL.Query().Get("service"),
		Pod:         r.URL.Query().Get("pod"),
		ProfileType: domain.ProfileType(r.URL.Query().Get("profile_type")),
		Start:       parseQueryTime(r.URL.Query().Get("start")),
		End:         parseQueryTime(r.URL.Query().Get("end")),
		Limit:       limit,
	}
}

func (h QueryHandlers) Deadlocks(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_deadlocks", func() (any, error) {
		return app.QueryDeadlocks(
			r.Context(),
			h.Threads,
			r.URL.Query().Get("namespace"),
			r.URL.Query().Get("service"),
			parseQueryLimit(r, app.DefaultDeadlockLimit, app.MaxDeadlockLimit),
		)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) TargetStatus(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_target_status", func() (any, error) {
		return app.QueryTargetStatus(
			r.Context(),
			h.Statuses,
			r.URL.Query().Get("namespace"),
			r.URL.Query().Get("service"),
			parseQueryTime(r.URL.Query().Get("start")),
			parseQueryTime(r.URL.Query().Get("end")),
			parseQueryLimit(r, app.DefaultTargetStatusLimit, app.MaxTargetStatusLimit),
		)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) Ingestion(w http.ResponseWriter, r *http.Request) {
	result, err := h.observe("java_profiler_http_query_ingestion", func() (any, error) {
		return app.QueryIngestionHealth(r.Context(), h.IngestionStore, h.Metrics)
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) observe(metricPrefix string, fn func() (any, error)) (any, error) {
	started := time.Now()
	result, err := fn()
	recordMetric(h.Metrics, metricPrefix+"_requests_total", 1)
	recordMetric(h.Metrics, metricPrefix+"_duration_seconds_total", time.Since(started).Seconds())
	if err != nil {
		recordMetric(h.Metrics, metricPrefix+"_errors_total", 1)
		return result, err
	}
	switch v := result.(type) {
	case []clickhouse.DeadlockEvent:
		recordMetric(h.Metrics, metricPrefix+"_rows_total", float64(len(v)))
	case []clickhouse.TargetStatus:
		recordMetric(h.Metrics, metricPrefix+"_rows_total", float64(len(v)))
	case app.ThreadDiagnosis:
		recordMetric(h.Metrics, metricPrefix+"_busy_threads_total", float64(len(v.BusyThreads)))
		recordMetric(h.Metrics, metricPrefix+"_slow_threads_total", float64(len(v.SlowThreads)))
		if v.Partial {
			recordMetric(h.Metrics, metricPrefix+"_partial_total", 1)
		}
	case app.IngestionHealth:
		recordMetric(h.Metrics, metricPrefix+"_rows_total", float64(len(v.Batches)))
	case []app.TopStackRow:
		recordMetric(h.Metrics, metricPrefix+"_rows_total", float64(len(v)))
	case app.AllocationSummary:
		recordMetric(h.Metrics, metricPrefix+"_paths_total", float64(len(v.TopPaths)))
		recordMetric(h.Metrics, metricPrefix+"_self_frames_total", float64(len(v.TopSelfFrames)))
		if v.Coverage.Partial {
			recordMetric(h.Metrics, metricPrefix+"_partial_total", 1)
		}
	case app.ServiceProfileSummary:
		recordMetric(h.Metrics, metricPrefix+"_rows_total", float64(len(v.Targets)))
		if v.Partial {
			recordMetric(h.Metrics, metricPrefix+"_partial_total", 1)
		}
	}
	return result, err
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeQueryError(w http.ResponseWriter, status int, code any, message, field, suggestedAction string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(queryErrorResponse{
		Code:            codeString(code),
		Message:         message,
		Field:           field,
		SuggestedAction: suggestedAction,
	})
}

func codeString(code any) string {
	switch v := code.(type) {
	case app.AllocationSummaryValidationCode:
		return string(v)
	case string:
		return v
	default:
		return "query_error"
	}
}

func parseQueryTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if unixSeconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC()
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseQueryLimit(r *http.Request, fallback, maximum int) int {
	return parseSpecificQueryLimit(r, "limit", fallback, maximum)
}

func parseSpecificQueryLimit(r *http.Request, name string, fallback, maximum int) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if name != "limit" {
		limit, err = strconv.Atoi(r.URL.Query().Get(name))
	}
	if err != nil {
		return fallback
	}
	if limit <= 0 {
		return fallback
	}
	if limit > maximum {
		return maximum
	}
	return limit
}
