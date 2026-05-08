package discovery

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/procfs"
)

type ProcessInfo struct {
	PID       int
	Command   string
	StartTime time.Time
	Root      string
}

type ProcessScanner struct {
	ProcRoot string
	Now      func() time.Time
}

func (s ProcessScanner) Scan() ([]ProcessInfo, error) {
	root := s.ProcRoot
	if root == "" {
		root = "/proc"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	fs, fsErr := procfs.NewFS(root)
	var out []ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		processRoot := filepath.Join(root, entry.Name())
		cmdline, _ := os.ReadFile(filepath.Join(processRoot, "cmdline"))
		command := strings.ReplaceAll(string(cmdline), "\x00", " ")
		if strings.TrimSpace(command) == "" {
			command = entry.Name()
		}
		startTime := now()
		if fsErr == nil {
			if proc, procErr := fs.Proc(pid); procErr == nil {
				if stat, statErr := proc.Stat(); statErr == nil {
					if startedAt, startedErr := stat.StartTime(); startedErr == nil {
						seconds, frac := math.Modf(startedAt)
						startTime = time.Unix(int64(seconds), int64(frac*float64(time.Second))).UTC()
					}
				}
			}
		}
		out = append(out, ProcessInfo{PID: pid, Command: strings.TrimSpace(command), StartTime: startTime, Root: processRoot})
	}
	return out, nil
}
