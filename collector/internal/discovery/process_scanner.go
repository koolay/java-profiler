package discovery

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	var out []ProcessInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		cmdline, _ := os.ReadFile(filepath.Join(root, entry.Name(), "cmdline"))
		command := strings.ReplaceAll(string(cmdline), "\x00", " ")
		if strings.TrimSpace(command) == "" {
			command = entry.Name()
		}
		out = append(out, ProcessInfo{PID: pid, Command: strings.TrimSpace(command), StartTime: now(), Root: filepath.Join(root, entry.Name())})
	}
	return out, nil
}
