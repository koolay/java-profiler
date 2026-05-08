package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHotSpotDetectorDetectsCompatibleJvmAndConflict(t *testing.T) {
	root := t.TempDir()
	data := "/usr/lib/jvm/java-17/lib/server/libjvm.so\n/tmp/libasyncProfiler.so\n"
	if err := os.WriteFile(filepath.Join(root, "maps"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got := (HotSpotDetector{}).Detect(root)
	if !got.HotSpotCompatible {
		t.Fatalf("expected HotSpot-compatible JVM")
	}
	if !got.Conflict {
		t.Fatalf("expected async-profiler conflict")
	}
}
