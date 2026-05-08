package profiler

import (
	"context"
	"fmt"
	"os/exec"
)

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, path string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, string(out))
	}
	return out, nil
}
