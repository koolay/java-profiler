package runtime

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/koolay/java-profiler/collector/internal/discovery"
	collectorMetrics "github.com/koolay/java-profiler/collector/internal/metrics"
	"github.com/koolay/java-profiler/collector/internal/pipeline"
	"github.com/koolay/java-profiler/collector/internal/policy"
	collectorStatus "github.com/koolay/java-profiler/collector/internal/status"
	"github.com/koolay/java-profiler/domain"
)

type Config struct {
	ProcRoot     string
	CollectorID  string
	BackendURL   string
	BackendToken string
	NodeName     string
	Cluster      string
	PollInterval time.Duration
}

type Runtime struct {
	scanner      discovery.ProcessScanner
	detector     discovery.HotSpotDetector
	statuses     *collectorStatus.Store
	exporter     *collectorMetrics.Exporter
	backend      pipeline.BackendClient
	collectorID  string
	nodeName     string
	cluster      string
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
		nodeName:     cfg.NodeName,
		cluster:      firstNonEmpty(cfg.Cluster, "local"),
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
	pods := r.discoverPods(ctx)
	compatible := 0
	conflicting := 0
	unsupported := 0
	for _, process := range processes {
		eligibility := r.detector.Detect(process.Root)
		pod, hasPod := pods[podUIDFromProcess(process.Root)]
		target := domain.TargetIdentity{
			Cluster:       r.cluster,
			Namespace:     namespaceFromCommand(process.Command),
			Workload:      workloadFromCommand(process.Command),
			Pod:           process.Command,
			Container:     "process",
			ProcessID:     process.PID,
			JVMStartTime:  process.StartTime,
			RuntimeVendor: eligibility.Vendor,
			Service:       serviceFromCommand(process.Command),
		}
		eval := policy.Evaluate(policy.Metadata{ObservedAt: started})
		if hasPod {
			target.Namespace = pod.Metadata.Namespace
			target.Workload = ownerName(pod)
			target.Pod = pod.Metadata.Name
			target.Container = firstContainerName(pod)
			target.Node = pod.Spec.NodeName
			target.PodUID = string(pod.Metadata.UID)
			target.Service = serviceName(pod)
			eval = policy.Evaluate(policy.Metadata{
				Annotations: pod.Metadata.Annotations,
				Labels:      pod.Metadata.Labels,
				StartedAt:   pod.Metadata.CreationTimestamp.Time,
				ObservedAt:  started,
			})
		}
		status := collectorStatus.TargetStatus{
			Target:   target,
			StatusAt: started,
			State:    eval.DesiredState,
			Reason:   eval.Reason,
			Message:  eval.Message,
		}
		switch {
		case eval.DesiredState == domain.TargetDesiredStateDisabled:
			status.State = domain.TargetDesiredStateDisabled
			status.Reason = eval.Reason
			status.Message = eval.Message
		case eligibility.Conflict:
			conflicting++
			status.Reason = domain.StatusReasonProfilerConflict
			status.Message = "async-profiler already present"
		case !eligibility.HotSpotCompatible:
			unsupported++
			status.State = domain.TargetDesiredStateUnsupported
			status.Reason = domain.StatusReasonUnsupportedJVM
			status.Message = eligibility.Reason
		default:
			compatible++
			if eval.Mode == domain.EnablementTemporary {
				status.State = domain.TargetDesiredStateTemporary
			} else {
				status.State = domain.TargetDesiredStateEnabled
			}
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
		statusBatchID := fmt.Sprintf("%s-status-%d", r.collectorID, started.UnixNano())
		statusBatch, err := pipeline.BuildTargetStatusBatch(statusBatchID, r.collectorID, r.statuses.Snapshot())
		if err != nil {
			r.exporter.Inc("java_profiler_collector_upload_failures")
			return err
		}
		statusClient := r.backend
		statusClient.URL = pipeline.TargetStatusURL(r.backend.URL)
		if err := statusClient.Upload(ctx, statusBatch); err != nil {
			r.exporter.Inc("java_profiler_collector_upload_failures")
			r.exporter.Inc("java_profiler_collector_upload_retryable")
			return err
		}
		batchID := fmt.Sprintf("%s-profile-%d", r.collectorID, started.UnixNano())
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

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

type podList struct {
	Items []podItem `json:"items"`
}

type podItem struct {
	Metadata podMetadata `json:"metadata"`
	Spec     podSpec     `json:"spec"`
}

type podMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	UID               string            `json:"uid"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	OwnerReferences   []ownerReference  `json:"ownerReferences"`
	CreationTimestamp timeWrapper       `json:"creationTimestamp"`
}

type timeWrapper struct {
	time.Time
}

func (t *timeWrapper) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

type ownerReference struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type podSpec struct {
	NodeName   string         `json:"nodeName"`
	Containers []podContainer `json:"containers"`
}

type podContainer struct {
	Name string `json:"name"`
}

func (r *Runtime) discoverPods(ctx context.Context) map[string]podItem {
	out := map[string]podItem{}
	nodeName := r.nodeName
	if nodeName == "" {
		return out
	}
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return out
	}
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return out
	}
	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return out
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}, Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://%s:%s/api/v1/pods?fieldSelector=spec.nodeName%%3D%s", host, port, nodeName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return out
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	resp, err := client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return out
	}
	var pods podList
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return out
	}
	for _, pod := range pods.Items {
		out[normalizeUID(pod.Metadata.UID)] = pod
	}
	return out
}

var podUIDPattern = regexp.MustCompile(`pod([0-9a-fA-F_-]{32,36})`)

func podUIDFromProcess(processRoot string) string {
	data, err := os.ReadFile(filepath.Join(processRoot, "cgroup"))
	if err != nil {
		return ""
	}
	match := podUIDPattern.FindStringSubmatch(string(data))
	if len(match) < 2 {
		return ""
	}
	return normalizeUID(match[1])
}

func normalizeUID(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "_", "-"))
}

func ownerName(pod podItem) string {
	for _, owner := range pod.Metadata.OwnerReferences {
		if owner.Name != "" {
			return owner.Name
		}
	}
	return pod.Metadata.Name
}

func firstContainerName(pod podItem) string {
	if len(pod.Spec.Containers) > 0 && pod.Spec.Containers[0].Name != "" {
		return pod.Spec.Containers[0].Name
	}
	return "container"
}

func serviceName(pod podItem) string {
	if value := pod.Metadata.Labels["app.kubernetes.io/name"]; value != "" {
		return value
	}
	if value := pod.Metadata.Labels["app"]; value != "" {
		return value
	}
	return ownerName(pod)
}
