package profiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/koolay/java-profiler/collector/internal/jfr"
	profiling "github.com/koolay/java-profiler/contracts/profiling"
	"github.com/koolay/java-profiler/domain"
)

type AttachController interface {
	LoadNativeAgent(ctx context.Context, pid int, agentPath string, args string) error
}

type Executor interface {
	Run(ctx context.Context, path string, args ...string) ([]byte, error)
}

type Config struct {
	ProcRoot     string
	AsprofPath   string
	LibraryPath  string
	TargetTmpDir string
	Now          func() time.Time
}

type Runner struct {
	cfg      Config
	attach   AttachController
	exec     Executor
	sessions map[string]session
}

type session struct {
	pid       int
	startedAt time.Time
	jfrPath   string
}

func NewRunner(cfg Config, attach AttachController) *Runner {
	if cfg.ProcRoot == "" {
		cfg.ProcRoot = "/proc"
	}
	if cfg.TargetTmpDir == "" {
		cfg.TargetTmpDir = "/tmp/java-profiler"
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	runner := &Runner{cfg: cfg, attach: attach, sessions: map[string]session{}}
	if cfg.AsprofPath != "" {
		runner.exec = OSExecutor{}
	}
	return runner
}

func (r *Runner) Collect(ctx context.Context, batchID string, target domain.TargetIdentity) ([]profiling.ProfileSample, error) {
	if r == nil || r.attach == nil {
		return nil, nil
	}
	if target.ProcessID <= 0 {
		return nil, fmt.Errorf("target process id is required")
	}
	if r.cfg.LibraryPath == "" {
		return nil, fmt.Errorf("async-profiler library path is required")
	}
	if err := r.stageLibrary(target.ProcessID); err != nil {
		return nil, err
	}
	nsPID, err := r.nsPID(target.ProcessID)
	if err != nil {
		return nil, err
	}
	key := target.Key()
	now := r.cfg.Now().UTC()
	prior, hasPrior := r.sessions[key]
	if !hasPrior || prior.pid != target.ProcessID {
		return nil, r.start(ctx, key, target.ProcessID, nsPID, now)
	}
	if err := r.stop(ctx, target.ProcessID, prior.jfrPath); err != nil {
		return nil, err
	}
	events, err := (jfr.Parser{}).ParseFile(r.hostRootPath(target.ProcessID, prior.jfrPath))
	if err != nil {
		_ = r.start(ctx, key, target.ProcessID, nsPID, now)
		return nil, err
	}
	_ = os.Remove(r.hostRootPath(target.ProcessID, prior.jfrPath))
	samples := jfr.NormalizeWindow(batchID, target, events, prior.startedAt, now)
	if err := r.start(ctx, key, target.ProcessID, nsPID, now); err != nil {
		return samples, err
	}
	return samples, nil
}

func (r *Runner) HasSession(target domain.TargetIdentity) bool {
	if r == nil {
		return false
	}
	session, ok := r.sessions[target.Key()]
	return ok && session.pid == target.ProcessID
}

func (r *Runner) stageLibrary(pid int) error {
	data, err := os.ReadFile(r.cfg.LibraryPath)
	if err != nil {
		return err
	}
	dir := r.hostRootPath(pid, r.cfg.TargetTmpDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeFileIfChanged(filepath.Join(dir, "libasyncProfiler.so"), data)
}

func writeFileIfChanged(path string, data []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}
	return os.WriteFile(path, data, 0o755)
}

func (r *Runner) start(ctx context.Context, key string, pid int, nsPID int, startedAt time.Time) error {
	jfrPath := r.jfrPath(nsPID)
	if r.exec != nil && r.cfg.AsprofPath != "" {
		if _, err := r.exec.Run(ctx, r.cfg.AsprofPath,
			"start",
			"-e", "itimer",
			"-i", "10ms",
			"--alloc", "512k",
			"--lock", "10us",
			"-o", "jfr",
			"-f", jfrPath,
			"--libpath", r.targetLibraryPath(),
			strconv.Itoa(pid),
		); err != nil {
			return err
		}
		r.sessions[key] = session{pid: pid, startedAt: startedAt, jfrPath: jfrPath}
		return nil
	}
	args := fmt.Sprintf("start,file=%s,jfr,event=itimer,interval=10ms,alloc=512k,lock=10us", jfrPath)
	if err := r.attach.LoadNativeAgent(ctx, pid, r.targetLibraryPath(), args); err != nil {
		return err
	}
	r.sessions[key] = session{pid: pid, startedAt: startedAt, jfrPath: jfrPath}
	return nil
}

func (r *Runner) stop(ctx context.Context, pid int, jfrPath string) error {
	if r.exec != nil && r.cfg.AsprofPath != "" {
		args := []string{"stop", "-o", "jfr", "--libpath", r.targetLibraryPath(), strconv.Itoa(pid)}
		if jfrPath != "" {
			args = []string{"stop", "-o", "jfr", "-f", jfrPath, "--libpath", r.targetLibraryPath(), strconv.Itoa(pid)}
		}
		_, err := r.exec.Run(ctx, r.cfg.AsprofPath, args...)
		return err
	}
	args := "stop"
	if jfrPath != "" {
		args = "stop,file=" + jfrPath + ",jfr"
	}
	return r.attach.LoadNativeAgent(ctx, pid, r.targetLibraryPath(), args)
}

func (r *Runner) hostRootPath(pid int, insidePath string) string {
	return filepath.Join(r.cfg.ProcRoot, fmt.Sprintf("%d", pid), "root", strings.TrimPrefix(insidePath, "/"))
}

func (r *Runner) targetLibraryPath() string {
	return filepath.ToSlash(filepath.Join(r.cfg.TargetTmpDir, "libasyncProfiler.so"))
}

func (r *Runner) jfrPath(nsPID int) string {
	return filepath.ToSlash(filepath.Join(r.cfg.TargetTmpDir, fmt.Sprintf("ap_%d.jfr", nsPID)))
}

func (r *Runner) nsPID(pid int) (int, error) {
	data, err := os.ReadFile(filepath.Join(r.cfg.ProcRoot, strconv.Itoa(pid), "status"))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "NSpid:" {
			continue
		}
		value := fields[len(fields)-1]
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("invalid NSpid value: %w", err)
		}
		return parsed, nil
	}
	return 0, fmt.Errorf("NSpid not found for pid %d", pid)
}

type HotSpotAttachController struct {
	ProcRoot          string
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
}

func (c HotSpotAttachController) LoadNativeAgent(ctx context.Context, pid int, agentPath string, args string) error {
	msg := strings.Join([]string{"1", "load", agentPath, "true", args}, "\x00") + "\x00"
	status, resp, err := c.sendCommand(ctx, pid, msg)
	if err != nil {
		return err
	}
	if status != '0' {
		return fmt.Errorf("load native agent failed: status=%c response=%s", status, resp)
	}
	return nil
}

func (c HotSpotAttachController) sendCommand(ctx context.Context, pid int, msg string) (byte, string, error) {
	conn, err := c.dial(ctx, pid)
	if err != nil {
		return 0, "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.requestTimeout()))
	if _, err := conn.Write([]byte(msg)); err != nil {
		return 0, "", err
	}
	status := make([]byte, 1)
	if _, err := io.ReadFull(conn, status); err != nil {
		return 0, "", err
	}
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, conn)
	return status[0], strings.TrimSpace(buf.String()), nil
}

func (c HotSpotAttachController) dial(ctx context.Context, pid int) (net.Conn, error) {
	nsPID, err := c.nsPID(pid)
	if err != nil {
		return nil, err
	}
	sockPath := c.procPath(pid, "root", "tmp", fmt.Sprintf(".java_pid%d", nsPID))
	attachFiles := []string{
		c.procPath(pid, "cwd", fmt.Sprintf(".attach_pid%d", nsPID)),
		c.procPath(pid, "root", "tmp", fmt.Sprintf(".attach_pid%d", nsPID)),
	}
	if !isSocket(sockPath) {
		createdFile := ""
		for _, attachFile := range attachFiles {
			err = os.WriteFile(attachFile, []byte(""), 0o660)
			if err != nil && !os.IsExist(err) {
				continue
			}
			createdFile = attachFile
			break
		}
		if createdFile == "" {
			return nil, fmt.Errorf("failed to create JVM attach trigger file: %w", err)
		}
		defer os.Remove(createdFile)
		if err := syscall.Kill(pid, syscall.SIGQUIT); err != nil {
			return nil, err
		}
		if err := c.waitForSocket(ctx, sockPath); err != nil {
			return nil, err
		}
	}
	return net.DialTimeout("unix", sockPath, c.connectionTimeout())
}

func (c HotSpotAttachController) nsPID(pid int) (int, error) {
	r := Runner{cfg: Config{ProcRoot: c.procRoot()}}
	return r.nsPID(pid)
}

func (c HotSpotAttachController) waitForSocket(ctx context.Context, path string) error {
	deadline := time.NewTimer(c.connectionTimeout())
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for JVM attach socket %s", path)
		case <-ticker.C:
			if isSocket(path) {
				return nil
			}
		}
	}
}

func (c HotSpotAttachController) procPath(pid int, parts ...string) string {
	all := append([]string{c.procRoot(), strconv.Itoa(pid)}, parts...)
	return filepath.Join(all...)
}

func (c HotSpotAttachController) procRoot() string {
	if c.ProcRoot != "" {
		return c.ProcRoot
	}
	return "/proc"
}

func (c HotSpotAttachController) connectionTimeout() time.Duration {
	if c.ConnectionTimeout > 0 {
		return c.ConnectionTimeout
	}
	return 5 * time.Second
}

func (c HotSpotAttachController) requestTimeout() time.Duration {
	if c.RequestTimeout > 0 {
		return c.RequestTimeout
	}
	return 10 * time.Second
}

func isSocket(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode()&os.ModeSocket != 0
}
