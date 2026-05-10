package profiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type SessionMarker struct {
	CollectorID string    `json:"CollectorID"`
	PID         int       `json:"PID"`
	StartedAt   time.Time `json:"StartedAt"`
	LibraryPath string    `json:"LibraryPath"`
}

func markerPath(procRoot string, pid int) string {
	return filepath.Join(procRoot, strconv.Itoa(pid), "root", "tmp", "java-profiler-session.json")
}

func WriteSessionMarker(procRoot string, pid int, marker SessionMarker) error {
	path := markerPath(procRoot, pid)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ReadSessionMarker(procRoot string, pid int) (SessionMarker, error) {
	data, err := os.ReadFile(markerPath(procRoot, pid))
	if err != nil {
		return SessionMarker{}, err
	}
	var marker SessionMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return SessionMarker{}, err
	}
	return marker, nil
}

func RemoveSessionMarker(procRoot string, pid int) error {
	if err := os.Remove(markerPath(procRoot, pid)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
