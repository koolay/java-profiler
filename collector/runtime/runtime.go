package runtime

import (
	"net/http"

	"github.com/koolay/java-profiler/collector/internal/metrics"
)

func NewMetricsHandler() http.Handler {
	exporter := metrics.NewExporter()
	exporter.Set("java_profiler_collector_up", 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(exporter.Snapshot()))
	})
	return mux
}
