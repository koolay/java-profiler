package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	collectorruntime "github.com/koolay/java-profiler/collector/runtime"
)

func main() {
	addr := os.Getenv("JAVA_PROFILER_METRICS_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	runtime := collectorruntime.NewCollector(collectorruntime.Config{
		ProcRoot:     firstNonEmpty(os.Getenv("JAVA_PROFILER_PROC_ROOT"), "/host/proc"),
		CollectorID:  collectorID(),
		BackendURL:   firstNonEmpty(os.Getenv("JAVA_PROFILER_COLLECTOR_BACKEND_URL"), os.Getenv("JAVA_PROFILER_BACKEND_URL")),
		BackendToken: firstNonEmpty(os.Getenv("JAVA_PROFILER_COLLECTOR_BACKEND_TOKEN"), os.Getenv("JAVA_PROFILER_BACKEND_TOKEN")),
		PollInterval: parseDuration("JAVA_PROFILER_COLLECTOR_INTERVAL", time.Minute),
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := runtime.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("collector runtime stopped: %v", err)
			stop()
		}
	}()
	log.Printf("java-profiler collector metrics listening on %s", addr)
	server := &http.Server{Addr: addr, Handler: collectorruntime.NewMetricsHandler(runtime.Exporter())}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func collectorID() string {
	if value := os.Getenv("JAVA_PROFILER_COLLECTOR_ID"); value != "" {
		return value
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "java-profiler-collector"
	}
	return host
}

func parseDuration(name string, fallback time.Duration) time.Duration {
	if value := os.Getenv(name); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
