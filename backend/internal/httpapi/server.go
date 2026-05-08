package httpapi

import (
	"context"
	"fmt"
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
	AllowInMemory bool
}

func NewServer(cfg ServerConfig, exporter *metrics.Exporter) (http.Handler, error) {
	var profiles app.ProfileQueryStore
	var ingestion app.IngestionStore
	if cfg.ClickHouseDSN != "" {
		sqlRepo, err := clickhouse.OpenSQLRepository(cfg.ClickHouseDSN)
		if err != nil {
			return nil, fmt.Errorf("open clickhouse repository: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := sqlRepo.Ping(ctx); err != nil {
			return nil, fmt.Errorf("ping clickhouse: %w", err)
		}
		if err := sqlRepo.ApplySchema(ctx); err != nil {
			return nil, fmt.Errorf("apply clickhouse schema: %w", err)
		}
		profiles = sqlRepo
		ingestion = sqlRepo
	} else if cfg.AllowInMemory {
		profiles = clickhouse.NewProfileRepository()
		ingestion = clickhouse.NewIngestionRepository()
	} else {
		return nil, fmt.Errorf("JAVA_PROFILER_CLICKHOUSE_DSN is required unless in-memory mode is explicitly enabled")
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
	return mux, nil
}

func NewServerFromEnv(exporter *metrics.Exporter) (http.Handler, error) {
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
		AllowInMemory: os.Getenv("JAVA_PROFILER_ALLOW_IN_MEMORY") == "true",
	}, exporter)
}
