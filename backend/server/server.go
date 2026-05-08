package server

import (
	"net/http"

	"github.com/koolay/java-profiler/backend/internal/httpapi"
	"github.com/koolay/java-profiler/backend/internal/metrics"
)

func NewFromEnv() (http.Handler, error) {
	return httpapi.NewServerFromEnv(metrics.NewExporter())
}
