package main

import (
	"log"
	"net/http"
	"os"

	"github.com/koolay/java-profiler/backend/server"
)

func main() {
	addr := os.Getenv("JAVA_PROFILER_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	handler := server.NewFromEnv()
	log.Printf("java-profiler backend listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
