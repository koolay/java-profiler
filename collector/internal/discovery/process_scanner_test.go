package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessScannerFindsNumericProcesses(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "123"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "123", "cmdline"), []byte("java\x00-jar\x00app.jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  0 0 0 0 0 0 0 0 0 0\nbtime 1000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "123", "stat"), []byte("123 (java) S 1 1 1 0 -1 4194560 0 0 0 0 0 0 0 0 20 0 1 0 200 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "not-pid"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := (ProcessScanner{ProcRoot: root, Now: func() time.Time { return time.Unix(1, 0) }}).Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PID != 123 || got[0].Command != "java -jar app.jar" {
		t.Fatalf("unexpected scan result: %+v", got)
	}
	if got[0].StartTime != time.Unix(1002, 0).UTC() {
		t.Fatalf("expected parsed start time, got %s", got[0].StartTime)
	}
}
