import { useMemo, useState } from "react";
import { getAllocationSummary, getFlamegraph, getIngestionHealth, getJVMEvents, getTargetStatus } from "../../api/client";
import type { AllocationSummary, FlamegraphResponse, IngestionHealth, JVMEventEvidence, TargetStatus } from "../../api/types";
import { useAPI } from "../../api/use-api";
import { Flamegraph } from "../../visualization/flamegraph";
import { buildMemoryPressureInsights } from "./memory-pressure-insights";

export function MemoryPressureView({ params }: { params: URLSearchParams }) {
  const service = params.get("service") || params.get("pod") || "service";
  const allocationParams = useMemo(() => withParam(params, "profile_type", "java_allocation_bytes"), [params]);
  const gcParams = useMemo(() => withParam(params, "event_type", "gc_pause"), [params]);
  const fallbackFlamegraph: FlamegraphResponse = { root: { name: service, value: 0, children: [] }, metadata: { partial: false } };
  const fallbackAllocation = emptyAllocationSummary(allocationParams);
  const fallbackGC: JVMEventEvidence = { events: [], partial: false };
  const fallbackStatus: TargetStatus[] = [];
  const fallbackIngestion: IngestionHealth = {
    totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
    batches: [],
    partial: false,
  };
  const [searchQuery, setSearchQuery] = useState("");

  const { data: statuses, error: statusError, loading: statusLoading } = useAPI(() => getTargetStatus(params), [params.toString()], fallbackStatus);
  const { data: ingestion, error: ingestionError, loading: ingestionLoading } = useAPI(() => getIngestionHealth(), [], fallbackIngestion);
  const { data: gcEvidence, error: gcError, loading: gcLoading } = useAPI(() => getJVMEvents(gcParams), [gcParams.toString()], fallbackGC);
  const { data: allocation, error: allocationError, loading: allocationLoading } = useAPI(() => getAllocationSummary(allocationParams), [allocationParams.toString()], fallbackAllocation);
  const { data: flamegraph, error: flamegraphError, loading: flamegraphLoading } = useAPI(() => getFlamegraph(allocationParams), [allocationParams.toString()], fallbackFlamegraph);

  const latestStatus = latestTargetStatus(statuses ?? []);
  const latestBatch = latestAcceptedProfileBatch(ingestion ?? fallbackIngestion);
  const gcEvents = gcEvidence?.events ?? [];
  const gcPauseTotal = gcEvents.reduce((sum, event) => sum + (event.duration_ns ?? 0), 0);
  const summary = allocation ?? fallbackAllocation;
  const insights = buildMemoryPressureInsights(summary, gcEvidence ?? fallbackGC);

  return (
    <section className="profile-analysis profile-analysis-wide memory-pressure-workflow" aria-label="Memory pressure investigation">
      <div className="section-heading">
        <div>
          <p className="eyebrow">OOM / memory pressure</p>
          <h2>Memory pressure investigation</h2>
          <p>Correlate target status, ingestion freshness, GC pauses, sampled allocation sources, and flame graph context.</p>
        </div>
      </div>

      <div className="gc-summary-strip memory-pressure-strip" aria-label="Memory pressure evidence">
        <EvidenceCard label="Target status" value={latestStatus ? statusTitle(latestStatus) : "No status"} detail={latestStatus?.message || "No target status evidence returned."} />
        <EvidenceCard label="Ingestion" value={`${ingestion?.totals.accepted ?? 0} accepted`} detail={latestBatch ? `Latest profile batch ${latestBatch.latest_at}` : "No accepted profile batches returned."} />
        <EvidenceCard label="GC pauses" value={`${gcEvents.length} GC pause${gcEvents.length === 1 ? "" : "s"}`} detail={`${formatDuration(gcPauseTotal)} total pause evidence`} />
        <EvidenceCard label="Sampled allocation" value={formatAllocationValue(summary.coverage.total_value, summary.coverage.value_unit)} detail={`${summary.coverage.scanned_samples} scanned samples`} />
      </div>

      {statusError && <p className="warning">Target status unavailable: {statusError}</p>}
      {ingestionError && <p className="warning">Ingestion evidence unavailable: {ingestionError}</p>}
      {gcError && <p className="warning">GC event evidence unavailable: {gcError}</p>}
      {allocationError && <p className="warning">Allocation summary unavailable: {allocationError}</p>}
      {flamegraphError && <p className="warning">Allocation flamegraph unavailable: {flamegraphError}</p>}
      {summary.coverage.partial && <p className="warning">Partial allocation evidence: some paths or frames were omitted from this response.</p>}

      <div className="allocation-insights memory-pressure-insights" aria-label="Memory pressure insights">
        {insights.length === 0 ? (
          <p className="muted">No memory-pressure insights returned for this scope.</p>
        ) : (
          insights.map((insight) => (
            <button className={`insight-row insight-row-${insight.severity}`} key={insight.code} type="button" onClick={() => insight.frame && setSearchQuery(insight.frame)}>
              <strong>{insightLabel(insight.code)}:</strong> {insight.message}
            </button>
          ))
        )}
      </div>

      {(statusLoading || ingestionLoading || gcLoading || allocationLoading || flamegraphLoading) && <p className="muted">Loading memory pressure evidence.</p>}
      <Flamegraph
        root={flamegraph?.root ?? fallbackFlamegraph.root}
        metadata={flamegraph?.metadata}
        profileType="java_allocation_bytes"
        searchQuery={searchQuery}
        onSearchQueryChange={setSearchQuery}
        emptyMessage="No allocation samples returned for this memory-pressure window."
        formatValue={(value) => formatAllocationValue(value, "bytes")}
        valueLabel="Allocated bytes"
        inspectorTotalLabel="Total Allocated"
        inspectorSelfLabel="Self Allocated"
        detailTotalPercentLabel="Total Allocated %"
        detailSelfPercentLabel="Self Allocated %"
      />
    </section>
  );
}

function EvidenceCard({ label, value, detail }: { label: string; value: string; detail: string }) {
  return (
    <div className="gc-summary-card">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function withParam(params: URLSearchParams, key: string, value: string) {
  const next = new URLSearchParams(params);
  next.set(key, value);
  return next;
}

function latestTargetStatus(statuses: TargetStatus[]) {
  return [...statuses].sort((a, b) => {
    const timeOrder = Date.parse(b.status_at ?? "") - Date.parse(a.status_at ?? "");
    if (Number.isFinite(timeOrder) && timeOrder !== 0) return timeOrder;
    return statusPriority(b) - statusPriority(a);
  })[0];
}

function statusPriority(status: TargetStatus) {
  if (status.reason === "oom_killed_seen") return 50;
  if (status.reason === "container_restarted" || status.reason === "profiling_window_after_restart") return 40;
  if (status.reason === "accepted") return 30;
  if (status.desired_state === "enabled" || status.desired_state === "temporary") return 20;
  return 0;
}

function latestAcceptedProfileBatch(ingestion: IngestionHealth) {
  return [...(ingestion.batches ?? [])]
    .filter((batch) => batch.batch_type === "profile" && batch.status === "accepted")
    .sort((a, b) => Date.parse(b.latest_at) - Date.parse(a.latest_at))[0];
}

function statusTitle(status: TargetStatus) {
  if (status.reason === "accepted") return "Profiling accepted";
  return humanize(status.reason || status.desired_state || "unknown");
}

function humanize(value: string) {
  return value
    .split(/[_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function insightLabel(code: string) {
  return humanize(code.replace(/^gc_/, "gc "));
}

function emptyAllocationSummary(params: URLSearchParams): AllocationSummary {
  return {
    schema_version: 1,
    requested_scope: scopeFromParams(params),
    effective_scope: scopeFromParams(params),
    coverage: {
      has_data: false,
      profile_type: "java_allocation_bytes",
      total_value: 0,
      value_unit: "bytes",
      scanned_samples: 0,
      returned_paths: 0,
      returned_self_frames: 0,
      omitted_paths_lower_bound: 0,
      omitted_nodes_lower_bound: 0,
      partial: false,
    },
    top_paths: [],
    top_self_frames: [],
    insights: [],
    limitations: [],
  };
}

function scopeFromParams(params: URLSearchParams) {
  return {
    namespace: params.get("namespace") ?? "",
    service: params.get("service") ?? "",
    pod: params.get("pod") ?? "",
    container: params.get("container") ?? "",
    jvm: params.get("jvm") ?? "",
    start: params.get("start") ?? "",
    end: params.get("end") ?? "",
  };
}

function formatAllocationValue(value: number, unit: string) {
  if (unit !== "bytes") return `${value.toLocaleString()} ${unit}`;
  if (value >= 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GiB`;
  if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KiB`;
  return `${value} B`;
}

function formatDuration(ns: number) {
  if (ns >= 1_000_000_000) return `${(ns / 1_000_000_000).toFixed(2)}s`;
  if (ns >= 1_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
  if (ns >= 1_000) return `${(ns / 1_000).toFixed(1)}us`;
  return `${ns}ns`;
}
