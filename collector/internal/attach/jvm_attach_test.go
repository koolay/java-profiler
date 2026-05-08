package attach

import (
	"testing"
	"time"
)

func TestPlanBuildsStartCommand(t *testing.T) {
	cmd, err := (Plan{LibraryPath: "/assets/libasyncProfiler.so", WorkDir: "/tmp/java-profiler"}).StartCommand(42, "cpu", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.PID != 42 || cmd.Event != "cpu" || cmd.OutputPath == "" {
		t.Fatalf("unexpected command: %+v", cmd)
	}
}
