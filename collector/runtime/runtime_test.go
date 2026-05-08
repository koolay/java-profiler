package runtime

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeScansProcessesAndExportsStatusMetrics(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "123")
	if err := os.Mkdir(proc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("btime 1000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proc, "cmdline"), []byte("java\x00-jar\x00app.jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proc, "stat"), []byte("123 (java) S 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 100 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proc, "maps"), []byte("/usr/lib/jvm/server/libjvm.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{ProcRoot: root, Now: func() time.Time { return time.Unix(2000, 0) }})
	if err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	r.MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "java_profiler_collector_discovered_processes 1") {
		t.Fatalf("missing discovered process metric:\n%s", body)
	}
	if !strings.Contains(body, "java_profiler_collector_eligible_jvms 1") {
		t.Fatalf("missing eligible JVM metric:\n%s", body)
	}
}
