package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/koolay/java-profiler/backend/internal/app"
	"github.com/koolay/java-profiler/domain"
)

type QueryHandlers struct {
	Profiles       app.ProfileQueryStore
	Threads        app.ThreadStore
	Statuses       app.TargetStatusQueryStore
	IngestionStore app.IngestionQueryStore
}

func (h QueryHandlers) Flamegraph(w http.ResponseWriter, r *http.Request) {
	result, err := (app.FlamegraphQuerier{Profiles: h.Profiles}).Query(r.Context(), app.FlamegraphQuery{
		Namespace:   r.URL.Query().Get("namespace"),
		Service:     r.URL.Query().Get("service"),
		Pod:         r.URL.Query().Get("pod"),
		ProfileType: domain.ProfileType(r.URL.Query().Get("profile_type")),
		Start:       parseQueryTime(r.URL.Query().Get("start")),
		End:         parseQueryTime(r.URL.Query().Get("end")),
		Limit:       1000,
		NodeLimit:   2048,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) ThreadDiagnosis(w http.ResponseWriter, r *http.Request) {
	result, err := app.QueryThreadDiagnosis(r.Context(), h.Threads, r.URL.Query().Get("namespace"), r.URL.Query().Get("service"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) Deadlocks(w http.ResponseWriter, r *http.Request) {
	result, err := app.QueryDeadlocks(r.Context(), h.Threads, r.URL.Query().Get("namespace"), r.URL.Query().Get("service"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) TargetStatus(w http.ResponseWriter, r *http.Request) {
	result, err := app.QueryTargetStatus(
		r.Context(),
		h.Statuses,
		r.URL.Query().Get("namespace"),
		r.URL.Query().Get("service"),
		parseQueryTime(r.URL.Query().Get("start")),
		parseQueryTime(r.URL.Query().Get("end")),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func (h QueryHandlers) Ingestion(w http.ResponseWriter, r *http.Request) {
	result, err := app.QueryIngestionHealth(r.Context(), h.IngestionStore)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, result)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
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
