package httpapi

import "github.com/koolay/java-profiler/backend/internal/metrics"

func recordMetric(exporter *metrics.Exporter, name string, value float64) {
	if exporter == nil {
		return
	}
	exporter.Add(name, value)
}
