package attach

import (
	"context"
	"fmt"
	"time"
)

type Command struct {
	PID        int
	Action     string
	Event      string
	OutputPath string
	Duration   time.Duration
}

type Controller interface {
	Start(ctx context.Context, cmd Command) error
	Stop(ctx context.Context, pid int) (string, error)
}

type Plan struct {
	LibraryPath string
	WorkDir     string
}

func (p Plan) StartCommand(pid int, event string, duration time.Duration) (Command, error) {
	if pid <= 0 {
		return Command{}, fmt.Errorf("pid must be positive")
	}
	if p.LibraryPath == "" || p.WorkDir == "" {
		return Command{}, fmt.Errorf("library path and work dir are required")
	}
	return Command{
		PID:        pid,
		Action:     "start",
		Event:      event,
		OutputPath: fmt.Sprintf("%s/java-profiler-%d-%s.jfr", p.WorkDir, pid, event),
		Duration:   duration,
	}, nil
}
