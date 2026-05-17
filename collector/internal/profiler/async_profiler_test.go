package profiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koolay/java-profiler/domain"
)

type recordedAttach struct {
	pid       int
	agentPath string
	args      string
}

type recordingAttachController struct {
	commands []recordedAttach
}

func (c *recordingAttachController) LoadNativeAgent(_ context.Context, pid int, agentPath string, args string) error {
	c.commands = append(c.commands, recordedAttach{pid: pid, agentPath: agentPath, args: args})
	return nil
}

type recordedCommand struct {
	path string
	args []string
}

type recordingExecutor struct {
	commands []recordedCommand
}

func (e *recordingExecutor) Run(_ context.Context, path string, args ...string) ([]byte, error) {
	e.commands = append(e.commands, recordedCommand{path: path, args: append([]string(nil), args...)})
	return []byte("ok"), nil
}

func TestRunnerUsesOfficialAsprofCLIWhenConfigured(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{
		ProcRoot:     procRoot,
		AsprofPath:   "/assets/asprof",
		LibraryPath:  libPath,
		TargetTmpDir: "/tmp/java-profiler",
		Now:          fixedTimes(time.Unix(100, 0), time.Unix(160, 0)),
	}, &recordingAttachController{})
	exec := &recordingExecutor{}
	runner.exec = exec
	target := domain.TargetIdentity{Cluster: "c", Namespace: "prod", Service: "jdk17-http-demo", Pod: "jdk17-http-demo-1", ProcessID: 42, JVMStartTime: time.Unix(1, 0)}

	if _, err := runner.Collect(context.Background(), "batch-1", target); err != nil {
		t.Fatal(err)
	}
	if len(exec.commands) != 1 {
		t.Fatalf("expected initial start asprof command, got %+v", exec.commands)
	}
	startArgs := strings.Join(exec.commands[0].args, " ")
	for _, want := range []string{"start -e itimer", "--wall 10ms", "-f /tmp/java-profiler/ap_7.jfr", "--libpath /tmp/java-profiler/libasyncProfiler.so"} {
		if !strings.Contains(startArgs, want) {
			t.Fatalf("expected start command to contain %q, got %s", want, startArgs)
		}
	}
	for _, unsafeDefault := range []string{"--alloc", "--lock"} {
		if strings.Contains(startArgs, unsafeDefault) {
			t.Fatalf("expected default profiler command not to contain %q, got %s", unsafeDefault, startArgs)
		}
	}
}

func TestRunnerCanOptIntoAllocationAndLockProfiling(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{
		ProcRoot:             procRoot,
		AsprofPath:           "/assets/asprof",
		LibraryPath:          libPath,
		TargetTmpDir:         "/tmp/java-profiler",
		AllocationAndLockJFR: true,
	}, &recordingAttachController{})
	exec := &recordingExecutor{}
	runner.exec = exec
	target := domain.TargetIdentity{Cluster: "c", Namespace: "prod", Service: "jdk17-http-demo", Pod: "jdk17-http-demo-1", ProcessID: 42, JVMStartTime: time.Unix(1, 0)}

	if _, err := runner.Collect(context.Background(), "batch-1", target); err != nil {
		t.Fatal(err)
	}
	startArgs := strings.Join(exec.commands[0].args, " ")
	for _, want := range []string{"--alloc 8m", "--lock 10us"} {
		if !strings.Contains(startArgs, want) {
			t.Fatalf("expected opt-in profiler command to contain %q, got %s", want, startArgs)
		}
	}
}

func TestRunnerCanDisableWallClockProfiling(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{
		ProcRoot:            procRoot,
		AsprofPath:          "/assets/asprof",
		LibraryPath:         libPath,
		TargetTmpDir:        "/tmp/java-profiler",
		DisableWallClockJFR: true,
	}, &recordingAttachController{})
	exec := &recordingExecutor{}
	runner.exec = exec
	target := domain.TargetIdentity{Cluster: "c", Namespace: "prod", Service: "jdk17-http-demo", Pod: "jdk17-http-demo-1", ProcessID: 42, JVMStartTime: time.Unix(1, 0)}

	if _, err := runner.Collect(context.Background(), "batch-1", target); err != nil {
		t.Fatal(err)
	}
	startArgs := strings.Join(exec.commands[0].args, " ")
	if strings.Contains(startArgs, "--wall") {
		t.Fatalf("expected wall clock to be disabled, got %s", startArgs)
	}
}

func TestRunnerUsesHotSpotAttachStopsParsesAndRestartsAsyncProfilerJFR(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	attach := &recordingAttachController{}
	runner := NewRunner(Config{
		ProcRoot:     procRoot,
		LibraryPath:  libPath,
		TargetTmpDir: "/tmp/java-profiler",
		Now:          fixedTimes(time.Unix(100, 0), time.Unix(160, 0)),
	}, attach)
	target := domain.TargetIdentity{Cluster: "c", Namespace: "prod", Service: "jdk17-http-demo", Pod: "jdk17-http-demo-1", ProcessID: 42, JVMStartTime: time.Unix(1, 0)}

	first, err := runner.Collect(context.Background(), "batch-1", target)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Samples) != 0 {
		t.Fatalf("first collection starts profiler only, got samples: %+v", first)
	}
	if len(attach.commands) != 1 {
		t.Fatalf("expected initial start attach command, got %+v", attach.commands)
	}
	start := attach.commands[0]
	if start.pid != 42 || start.agentPath != "/tmp/java-profiler/libasyncProfiler.so" || !strings.HasPrefix(start.args, "start,file=/tmp/java-profiler/ap_7.jfr,jfr,event=itimer,interval=10ms") || !strings.Contains(start.args, "wall=10ms") || strings.Contains(start.args, "alloc=") || strings.Contains(start.args, "lock=") {
		t.Fatalf("expected Coroot-style native agent start command, got %+v", start)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "tmp", "java-profiler", "libasyncProfiler.so")); err != nil {
		t.Fatalf("expected profiler library staged into target root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetRoot, "tmp", "java-profiler", "ap_7.jfr"), []byte("alloc_bytes|4096|com.ebpfjava.examples.httpdemo.DemoHttpService.allocateObjects;java.lang.Thread.run\nexecution_sample|7|com.ebpfjava.examples.httpdemo.DemoHttpService.burnCpu\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := runner.Collect(context.Background(), "batch-2", target)
	if err != nil {
		t.Fatal(err)
	}
	samples := result.Samples
	if len(samples) != 2 {
		t.Fatalf("expected parsed JFR samples, got %+v", samples)
	}
	if result.RawSampleCount != 2 {
		t.Fatalf("raw sample count = %d", result.RawSampleCount)
	}
	if samples[0].Frames[0] != "com.ebpfjava.examples.httpdemo.DemoHttpService.allocateObjects" || samples[0].Target.Service != "jdk17-http-demo" {
		t.Fatalf("expected real JFR stack and target identity, got %+v", samples[0])
	}
	if len(attach.commands) != 3 {
		t.Fatalf("expected start and stop/restart commands, got %+v", attach.commands)
	}
	if attach.commands[1].args != "stop,file=/tmp/java-profiler/ap_7.jfr,jfr" {
		t.Fatalf("expected stop command, got %+v", attach.commands[1])
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "tmp", "java-profiler", "ap_7.jfr")); !os.IsNotExist(err) {
		t.Fatalf("expected consumed JFR removed, stat err=%v", err)
	}
}

func TestRunnerDoesNotRewriteLoadedProfilerLibraryWhenUnchanged(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	targetRoot := filepath.Join(procRoot, "42", "root")
	if err := os.MkdirAll(filepath.Join(targetRoot, "tmp", "java-profiler"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(procRoot, "42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "42", "status"), []byte("Name:\tjava\nNSpid:\t4242\t7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(root, "libasyncProfiler.so")
	if err := os.WriteFile(libPath, []byte("native-profiler"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(targetRoot, "tmp", "java-profiler", "libasyncProfiler.so")
	if err := os.WriteFile(staged, []byte("native-profiler"), 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(Config{ProcRoot: procRoot, LibraryPath: libPath, TargetTmpDir: "/tmp/java-profiler"}, &recordingAttachController{})
	if err := runner.stageLibrary(42); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("expected unchanged profiler library not to be rewritten: before=%s after=%s", before.ModTime(), after.ModTime())
	}
}

func fixedTimes(values ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		if i >= len(values) {
			return values[len(values)-1]
		}
		value := values[i]
		i++
		return value
	}
}
