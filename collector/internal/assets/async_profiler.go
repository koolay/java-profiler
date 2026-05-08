package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

type AsyncProfilerAsset struct {
	Path          string
	Arch          string
	SHA256        string
	WritableMount string
}

func (a AsyncProfilerAsset) Verify() error {
	if a.Path == "" || a.SHA256 == "" {
		return fmt.Errorf("asset path and sha256 are required")
	}
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != a.SHA256 {
		return fmt.Errorf("asset checksum mismatch: got %s want %s", got, a.SHA256)
	}
	return nil
}
