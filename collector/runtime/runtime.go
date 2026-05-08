package runtime

import (
	"context"
	"net/http"
	"time"

	"github.com/koolay/java-profiler/collector/internal/discovery"
	"github.com/koolay/java-profiler/collector/internal/metrics"
	"github.com/koolay/java-profiler/collector/internal/pipeline"
	"github.com/koolay/java-profiler/domain"
)

type Config struct {
	ProcRoot   string
	BackendURL string
	Token      string
	Now        func() time.Time
}

type Runtime struct {
	cfg      Config
	exporter *metrics.Exporter
	scanner  discovery.ProcessScanner
	detector discovery.HotSpotDetector
	buffer   *pipeline.LocalBuffer
	client   pipeline.BackendClient
}

func New(cfg Config) *Runtime {
	exporter := metrics.NewExporter()
	exporter.Set("java_profiler_collector_up", 1)
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Runtime{
		cfg:      cfg,
		exporter: exporter,
		scanner:  discovery.ProcessScanner{ProcRoot: cfg.ProcRoot, Now: cfg.Now},
		detector: discovery.HotSpotDetector{},
		buffer:   pipeline.NewLocalBuffer(16<<20, 128),
		client:   pipeline.BackendClient{URL: cfg.BackendURL, Token: cfg.Token},
	}
}

func NewMetricsHandler() http.Handler {
	return New(Config{}).MetricsHandler()
}

func (r *Runtime) MetricsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(r.exporter.Snapshot()))
	})
	return mux
}

func (r *Runtime) RunOnce(ctx context.Context) error {
	processes, err := r.scanner.Scan()
	if err != nil {
		r.exporter.Set("java_profiler_collector_scan_errors_total", 1)
		return err
	}
	var eligible, skipped float64
	for _, process := range processes {
		eligibility := r.detector.Detect(process.Root)
		switch {
		case eligibility.Conflict:
			skipped++
			r.recordTargetStatus(process, domain.TargetDesiredStateDisabled, domain.StatusReasonProfilerConflict, "async-profiler conflict detected")
		case !eligibility.HotSpotCompatible:
			skipped++
			r.recordTargetStatus(process, domain.TargetDesiredStateUnsupported, domain.StatusReasonUnsupportedJVM, eligibility.Reason)
		default:
			eligible++
			r.recordTargetStatus(process, domain.TargetDesiredStateEnabled, domain.StatusReasonAccepted, "HotSpot-compatible JVM discovered")
		}
	}
	stats := r.buffer.Stats()
	r.exporter.Set("java_profiler_collector_discovered_processes", float64(len(processes)))
	r.exporter.Set("java_profiler_collector_eligible_jvms", eligible)
	r.exporter.Set("java_profiler_collector_skipped_jvms", skipped)
	r.exporter.Set("java_profiler_collector_buffered_batches", float64(stats.CurrentBatches))
	r.exporter.Set("java_profiler_collector_dropped_batches_total", float64(stats.DroppedBatches))
	return ctx.Err()
}

func (r *Runtime) recordTargetStatus(process discovery.ProcessInfo, desired domain.TargetDesiredState, reason domain.StatusReason, message string) {
	if message == "" {
		message = string(reason)
	}
	r.exporter.Inc("java_profiler_collector_target_status_" + string(reason))
	if desired == domain.TargetDesiredStateEnabled || desired == domain.TargetDesiredStateTemporary {
		r.exporter.Set("java_profiler_collector_last_enabled_jvm_start_unix", float64(process.StartTime.Unix()))
	}
}
