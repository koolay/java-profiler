import { useMemo } from "react";
import { getFlamegraph, getJVMEvents } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse, JVMEventEvidence } from "../../api/types";
import { HotCodeView } from "../cpu/hot-code-view";

export function GCView({ params }: { params: URLSearchParams }) {
  const gcParams = new URLSearchParams(params);
  gcParams.set("event_type", "gc_pause");
  const allocationParams = new URLSearchParams(params);
  allocationParams.set("profile_type", "java_allocation_bytes");
  const fallbackEvents: JVMEventEvidence = { events: [], partial: false };
  const fallbackProfile: FlamegraphResponse = { root: { name: params.get("service") ?? "service", value: 0, children: [] }, metadata: { partial: false } };
  const { data: events, error: eventsError } = useAPI(() => getJVMEvents(gcParams), [gcParams.toString()], fallbackEvents);
  const { data: allocation, error: allocationError } = useAPI(() => getFlamegraph(allocationParams), [allocationParams.toString()], fallbackProfile);
  const eventRows = events?.events ?? [];
  const summary = useMemo(() => summarizeEvents(eventRows), [eventRows]);

  return (
    <section className="gc-evidence" aria-label="GC evidence">
      <div className="profile-toolbar profile-toolbar-tight">
        <div>
          <h2>GC pauses</h2>
          <p>GC evidence is JVM-scoped and filtered by the same namespace, service, Pod, and time range as allocation samples.</p>
        </div>
      </div>
      {eventsError && <p className="warning">GC event evidence unavailable: {eventsError}</p>}
      {allocationError && <p className="warning">Allocation correlation unavailable: {allocationError}</p>}
      <div className="gc-summary-strip" aria-label="GC summary">
        <div className="gc-summary-card">
          <span>GC events</span>
          <strong>{summary.count}</strong>
        </div>
        <div className="gc-summary-card">
          <span>Total pause</span>
          <strong>{formatDuration(summary.totalDurationNs)}</strong>
        </div>
        <div className="gc-summary-card">
          <span>Max pause</span>
          <strong>{formatDuration(summary.maxDurationNs)}</strong>
        </div>
        <div className="gc-summary-card">
          <span>Average pause</span>
          <strong>{formatDuration(summary.averageDurationNs)}</strong>
        </div>
      </div>
      <div className="gc-event-list">
        {eventRows.length === 0 ? (
          <p className="muted">No GC pause event evidence in this range.</p>
        ) : (
          eventRows.map((event) => (
            <article className="gc-event-row" key={event.event_id}>
              <div>
                <strong>{formatDuration(event.duration_ns)}</strong>
                <span>{event.collector || "JVM GC"} · {event.action || event.event_type}</span>
              </div>
              <div>
                <span>{new Date(event.event_at).toLocaleString()}</span>
                <span>{event.cause || "cause unavailable"}</span>
              </div>
            </article>
          ))
        )}
      </div>
      <HotCodeView
        root={allocation?.root ?? fallbackProfile.root}
        metadata={allocation?.metadata}
        analysisLabel="GC allocation correlation analysis"
        profileType="java_allocation_bytes"
        title="Allocation correlation"
        description="Allocation flamegraph is shown beside GC pauses to expose Java allocation pressure in the same selected window."
        valueLabel="Allocated bytes"
        selfColumnLabel="Self Allocated"
        totalColumnLabel="Total Allocated"
      />
    </section>
  );
}

function summarizeEvents(events: JVMEventEvidence["events"]) {
  const count = events.length;
  const totalDurationNs = events.reduce((sum, event) => sum + Math.max(0, event.duration_ns), 0);
  const maxDurationNs = events.reduce((max, event) => Math.max(max, Math.max(0, event.duration_ns)), 0);
  const averageDurationNs = count > 0 ? totalDurationNs / count : 0;
  return { count, totalDurationNs, maxDurationNs, averageDurationNs };
}

function formatDuration(ns: number) {
  if (ns >= 1_000_000_000) return `${(ns / 1_000_000_000).toFixed(2)} s`;
  if (ns >= 1_000_000) return `${(ns / 1_000_000).toFixed(1)} ms`;
  if (ns >= 1_000) return `${(ns / 1_000).toFixed(1)} us`;
  return `${ns} ns`;
}
