package runtime

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/koolay/java-profiler/collector/internal/discovery"
	collectorMetrics "github.com/koolay/java-profiler/collector/internal/metrics"
	"github.com/koolay/java-profiler/collector/internal/pipeline"
	collectorStatus "github.com/koolay/java-profiler/collector/internal/status"
	"github.com/koolay/java-profiler/domain"
)

type Config struct {
	ProcRoot     string
	CollectorID  string
	BackendURL   string
	BackendToken string
	PollInterval time.Duration
}

type Runtime struct {
	scanner      discovery.ProcessScanner
	detector     discovery.HotSpotDetector
	statuses     *collectorStatus.Store
	exporter     *collectorMetrics.Exporter
	backend      pipeline.BackendClient
	collectorID  string
	pollInterval time.Duration
}

func NewCollector(cfg Config) *Runtime {
	procRoot := cfg.ProcRoot
	if procRoot == "" {
		procRoot = "/proc"
	}
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	collectorID := cfg.CollectorID
	if collectorID == "" {
		collectorID = "java-profiler-collector"
	}
	exporter := collectorMetrics.NewExporter()
	exporter.Set("java_profiler_collector_up", 1)
	return &Runtime{
		scanner:      discovery.ProcessScanner{ProcRoot: procRoot},
		detector:     discovery.HotSpotDetector{},
		statuses:     collectorStatus.NewStore(),
		exporter:     exporter,
		backend:      pipeline.BackendClient{URL: cfg.BackendURL, Token: cfg.BackendToken},
		collectorID:  collectorID,
		pollInterval: interval,
	}
}

func (r *Runtime) Exporter() *collectorMetrics.Exporter {
	return r.exporter
}

func (r *Runtime) MetricsHandler() http.Handler {
	return NewMetricsHandler(r.exporter)
}

func (r *Runtime) Statuses() []collectorStatus.TargetStatus {
	return r.statuses.Snapshot()
}

func (r *Runtime) Run(ctx context.Context) error {
	_ = r.ScanOnce(ctx)
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.ScanOnce(ctx); err != nil {
				r.exporter.Inc("java_profiler_collector_scan_errors")
			}
		}
	}
}

func (r *Runtime) ScanOnce(ctx context.Context) error {
	started := time.Now().UTC()
	processes, err := r.scanner.Scan()
	if err != nil {
		r.exporter.Inc("java_profiler_collector_scan_errors")
		return err
	}
	compatible := 0
	conflicting := 0
	unsupported := 0
	for _, process := range processes {
		eligibility := r.detector.Detect(process.Root)
		target := domain.TargetIdentity{
			Namespace:     namespaceFromCommand(process.Command),
			Workload:      workloadFromCommand(process.Command),
			Pod:           process.Command,
			Container:     "collector",
			ProcessID:     process.PID,
			JVMStartTime:  process.StartTime,
			RuntimeVendor: eligibility.Vendor,
			Service:       serviceFromCommand(process.Command),
		}
		status := collectorStatus.TargetStatus{
			Target:   target,
			StatusAt: started,
			State:    domain.TargetDesiredStateDisabled,
			Reason:   domain.StatusReasonUnsupportedJVM,
			Message:  "unsupported JVM",
		}
		switch {
		case eligibility.Conflict:
			conflicting++
			status.Reason = domain.StatusReasonProfilerConflict
			status.Message = "async-profiler already present"
		case !eligibility.HotSpotCompatible:
			unsupported++
			status.Message = eligibility.Reason
		default:
			compatible++
			status.State = domain.TargetDesiredStateEnabled
			status.Reason = domain.StatusReasonAccepted
			status.Message = "HotSpot-compatible JVM discovered"
		}
		r.statuses.Set(status)
		r.exporter.Inc("java_profiler_collector_target_status_" + string(status.Reason))
	}
	r.exporter.Set("java_profiler_collector_discovered_processes", float64(len(processes)))
	r.exporter.Set("java_profiler_collector_compatible_processes", float64(compatible))
	r.exporter.Set("java_profiler_collector_conflicting_processes", float64(conflicting))
	r.exporter.Set("java_profiler_collector_unsupported_processes", float64(unsupported))
	r.exporter.Set("java_profiler_collector_status_entries", float64(len(r.statuses.Snapshot())))
	r.exporter.Set("java_profiler_collector_last_scan_unix", float64(started.Unix()))

	if r.backend.URL != "" {
		batchID := fmt.Sprintf("%s-%d", r.collectorID, started.UnixNano())
		batch, err := pipeline.BuildProfileBatch(batchID, r.collectorID, nil)
		if err != nil {
			r.exporter.Inc("java_profiler_collector_upload_failures")
			return err
		}
		if err := r.backend.Upload(ctx, batch); err != nil {
			r.exporter.Inc("java_profiler_collector_upload_failures")
			r.exporter.Inc("java_profiler_collector_upload_retryable")
			return err
		}
		r.exporter.Inc("java_profiler_collector_upload_success")
	}
	return nil
}

func NewMetricsHandler(exporter *collectorMetrics.Exporter) http.Handler {
	if exporter == nil {
		exporter = collectorMetrics.NewExporter()
		exporter.Set("java_profiler_collector_up", 1)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(exporter.Snapshot()))
	})
	return mux
}

func namespaceFromCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "host"
	}
	if strings.Contains(fields[0], "java") {
		return "default"
	}
	return "host"
}

func workloadFromCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "process"
	}
	base := filepath.Base(fields[0])
	if base == "" {
		return "process"
	}
	return base
}

func serviceFromCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "process"
	}
	base := filepath.Base(fields[0])
	if strings.HasPrefix(base, "java") {
		return "java"
	}
	return base
}
