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
}
