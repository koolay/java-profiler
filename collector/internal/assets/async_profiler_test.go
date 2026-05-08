package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestAsyncProfilerAssetVerifyChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "libasyncProfiler.so")
	data := []byte("asset")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if err := (AsyncProfilerAsset{Path: path, SHA256: hex.EncodeToString(sum[:])}).Verify(); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}
