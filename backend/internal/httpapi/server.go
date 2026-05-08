package httpapi

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/koolay/java-profiler/backend/internal/app"
	"github.com/koolay/java-profiler/backend/internal/clickhouse"
	"github.com/koolay/java-profiler/backend/internal/metrics"
)

type ServerConfig struct {
	Auth          AuthConfig
	ClickHouseDSN string
}

func NewServer(cfg ServerConfig, exporter *metrics.Exporter) http.Handler {
	profileRepo := clickhouse.NewProfileRepository()
	ingestionRepo := clickhouse.NewIngestionRepository()
	var profiles app.ProfileQueryStore = profileRepo
	var ingestion app.IngestionStore = ingestionRepo
	if cfg.ClickHouseDSN != "" {
		if sqlRepo, err := clickhouse.OpenSQLRepository(cfg.ClickHouseDSN); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := sqlRepo.Ping(ctx); err == nil {
				_ = sqlRepo.ApplySchema(ctx)
				profiles = sqlRepo
				ingestion = sqlRepo
			}
			cancel()
		}
	}
	threadRepo := clickhouse.NewThreadRepository()
	statusRepo := clickhouse.NewStatusRepository()
	handlers := IngestHandlers{
		Profiles: app.ProfileBatchIngestor{Profiles: profiles, Ingestion: ingestion},
	}
	queryHandlers := QueryHandlers{Profiles: profiles, Threads: threadRepo, Statuses: statusRepo}
	mux := http.NewServeMux()
	mux.Handle("/api/collector/v1/profile-batches", RequireCollectorAuth(cfg.Auth, http.HandlerFunc(handlers.ProfileBatch)))
	mux.Handle("/api/ui/v1/flamegraph", RequireUIAuth(cfg.Auth, http.HandlerFunc(queryHandlers.Flamegraph)))
	mux.Handle("/api/ui/v1/thread-diagnosis", RequireUIAuth(cfg.Auth, http.HandlerFunc(queryHandlers.ThreadDiagnosis)))
	mux.Handle("/api/ui/v1/deadlocks", RequireUIAuth(cfg.Auth, http.HandlerFunc(queryHandlers.Deadlocks)))
	mux.Handle("/api/ui/v1/target-status", RequireUIAuth(cfg.Auth, http.HandlerFunc(queryHandlers.TargetStatus)))
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(exporter.Snapshot()))
	})
	return mux
}

func NewServerFromEnv(exporter *metrics.Exporter) http.Handler {
	collectorToken := os.Getenv("JAVA_PROFILER_COLLECTOR_TOKEN")
	if collectorToken == "" {
		collectorToken = os.Getenv("JAVA_PROFILER_AUTH_TOKEN")
	}
	uiToken := os.Getenv("JAVA_PROFILER_UI_TOKEN")
	if uiToken == "" {
		uiToken = os.Getenv("JAVA_PROFILER_AUTH_TOKEN")
	}
	return NewServer(ServerConfig{
		Auth: AuthConfig{
			CollectorToken: collectorToken,
			UIToken:        uiToken,
			RequireTLS:     os.Getenv("JAVA_PROFILER_REQUIRE_TLS") == "true",
		},
		ClickHouseDSN: os.Getenv("JAVA_PROFILER_CLICKHOUSE_DSN"),
	}, exporter)
}
