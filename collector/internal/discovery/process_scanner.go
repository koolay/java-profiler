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
	ProcRoot   string
	Now        func() time.Time
	ClockTicks int64
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
	bootTime := readBootTime(root)
	clockTicks := s.ClockTicks
	if clockTicks <= 0 {
		clockTicks = 100
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
		processRoot := filepath.Join(root, entry.Name())
		startTime := processStartTime(processRoot, bootTime, clockTicks)
		if startTime.IsZero() {
			startTime = now()
		}
		out = append(out, ProcessInfo{PID: pid, Command: strings.TrimSpace(command), StartTime: startTime, Root: processRoot})
	}
	return out, nil
}

func readBootTime(procRoot string) time.Time {
	data, err := os.ReadFile(filepath.Join(procRoot, "stat"))
	if err != nil {
		return time.Time{}
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "btime" {
			seconds, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil {
				return time.Unix(seconds, 0).UTC()
			}
		}
	}
	return time.Time{}
}

func processStartTime(processRoot string, bootTime time.Time, clockTicks int64) time.Time {
	if bootTime.IsZero() || clockTicks <= 0 {
		return time.Time{}
	}
	data, err := os.ReadFile(filepath.Join(processRoot, "stat"))
	if err != nil {
		return time.Time{}
	}
	text := string(data)
	closeParen := strings.LastIndex(text, ")")
	if closeParen < 0 || closeParen+2 >= len(text) {
		return time.Time{}
	}
	fields := strings.Fields(text[closeParen+2:])
	if len(fields) < 20 {
		return time.Time{}
	}
	startTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return bootTime.Add(time.Duration(startTicks) * time.Second / time.Duration(clockTicks)).UTC()
}
