package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	collectorruntime "github.com/koolay/java-profiler/collector/runtime"
)

func main() {
	addr := os.Getenv("JAVA_PROFILER_METRICS_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	runtime := collectorruntime.New(collectorruntime.Config{
		ProcRoot:   firstNonEmpty(os.Getenv("JAVA_PROFILER_PROC_ROOT"), "/host/proc"),
		BackendURL: os.Getenv("JAVA_PROFILER_BACKEND_URL"),
		Token:      os.Getenv("JAVA_PROFILER_BACKEND_TOKEN"),
	})
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if err := runtime.RunOnce(context.Background()); err != nil {
				log.Printf("collector runtime tick failed: %v", err)
			}
			<-ticker.C
		}
	}()
	log.Printf("java-profiler collector metrics listening on %s", addr)
	if err := http.ListenAndServe(addr, runtime.MetricsHandler()); err != nil {
		log.Fatal(err)
	}
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
