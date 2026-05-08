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
	Auth                AuthConfig
	ClickHouseDSN       string
	AllowInMemory       bool
	StorageReadyTimeout time.Duration
}

func NewServer(cfg ServerConfig, exporter *metrics.Exporter) (http.Handler, error) {
	var profiles app.ProfileQueryStore
	var ingestion app.IngestionStore
	var statuses app.TargetStatusQueryStore
	var statusIngest app.TargetStatusStore
	if cfg.ClickHouseDSN != "" {
		sqlRepo, err := clickhouse.OpenSQLRepository(cfg.ClickHouseDSN)
		if err != nil {
			return nil, fmt.Errorf("open clickhouse repository: %w", err)
		}
		readyTimeout := cfg.StorageReadyTimeout
		if readyTimeout <= 0 {
			readyTimeout = time.Minute
		}
		if err := waitForStorage(context.Background(), sqlRepo, readyTimeout); err != nil {
			return nil, err
		}
		profiles = sqlRepo
		ingestion = sqlRepo
		statuses = sqlRepo
		statusIngest = sqlRepo
	} else if cfg.AllowInMemory {
		profiles = clickhouse.NewProfileRepository()
		ingestion = clickhouse.NewIngestionRepository()
		statusRepo := clickhouse.NewStatusRepository()
		statuses = statusRepo
		statusIngest = statusRepo
	} else {
		return nil, fmt.Errorf("JAVA_PROFILER_CLICKHOUSE_DSN is required unless in-memory mode is explicitly enabled")
	}
	threadRepo := clickhouse.NewThreadRepository()
	handlers := IngestHandlers{
		Profiles:       app.ProfileBatchIngestor{Profiles: profiles, Ingestion: ingestion},
		TargetStatuses: app.TargetStatusIngestor{Statuses: statusIngest, Ingestion: ingestion},
	}
	queryHandlers := QueryHandlers{Profiles: profiles, Threads: threadRepo, Statuses: statuses}
	mux := http.NewServeMux()
	mux.Handle("/api/collector/v1/profile-batches", RequireCollectorAuth(cfg.Auth, http.HandlerFunc(handlers.ProfileBatch)))
	mux.Handle("/api/collector/v1/target-status-batches", RequireCollectorAuth(cfg.Auth, http.HandlerFunc(handlers.TargetStatusBatch)))
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

type schemaRepository interface {
	Ping(ctx context.Context) error
	ApplySchema(ctx context.Context) error
}

func waitForStorage(ctx context.Context, repo schemaRepository, timeout time.Duration) error {
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastErr error
	for {
		attemptCtx, attemptCancel := context.WithTimeout(deadline, 10*time.Second)
		err := repo.Ping(attemptCtx)
		if err == nil {
			err = repo.ApplySchema(attemptCtx)
		}
		attemptCancel()
		if err == nil {
			return nil
		}
		lastErr = err

		select {
		case <-deadline.Done():
			return fmt.Errorf("initialize clickhouse storage: %w", lastErr)
		case <-ticker.C:
		}
	}
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
