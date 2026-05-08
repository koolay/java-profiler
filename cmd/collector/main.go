package main

import (
	"log"
	"net/http"
	"os"

	collectorruntime "github.com/koolay/java-profiler/collector/runtime"
)

func main() {
	addr := os.Getenv("JAVA_PROFILER_METRICS_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	log.Printf("java-profiler collector metrics listening on %s", addr)
	if err := http.ListenAndServe(addr, collectorruntime.NewMetricsHandler()); err != nil {
		log.Fatal(err)
	}
}
