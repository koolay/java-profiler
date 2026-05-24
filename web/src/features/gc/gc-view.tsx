import { useMemo } from "react";
import { getFlamegraph, getJVMEvents } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { FlamegraphResponse, JVMEvent, JVMEventEvidence } from "../../api/types";
import { HotCodeView } from "../cpu/hot-code-view";

const maxVisibleEvents = 12;

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
  const profileWindow = useMemo(() => getProfileWindow(params), [params]);
  const summary = useMemo(() => summarizeEvents(eventRows, profileWindow?.durationMs), [eventRows, profileWindow?.durationMs]);
  const rankedEvents = useMemo(() => rankEvents(eventRows).slice(0, maxVisibleEvents), [eventRows]);

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
        <div className="gc-summary-card gc-summary-card-primary">
          <span>GC events</span>
          <strong>{summary.count}</strong>
          <small>{summary.eventRateLabel}</small>
        </div>
        <div className="gc-summary-card">
          <span>Total pause</span>
          <strong>{formatDuration(summary.totalDurationNs)}</strong>
          <small>{summary.pauseBudgetLabel}</small>
        </div>
        <div className="gc-summary-card">
          <span>Longest pause</span>
          <strong>{formatDuration(summary.maxDurationNs)}</strong>
          <small>{summary.severityLabel}</small>
        </div>
        <div className="gc-summary-card">
          <span>P95 pause</span>
          <strong>{formatDuration(summary.p95DurationNs)}</strong>
          <small>{summary.dominantCause}</small>
        </div>
      </div>
      <div className={`gc-brief gc-brief-${summary.severity}`} aria-label="GC interpretation">
        <div>
          <span>{summary.severityLabel}</span>
          <strong>{summary.interpretation}</strong>
        </div>
        <p>
          Sorted by pause duration so incident responders see the worst JVM stop-the-world evidence first. Allocation correlation below shows the code paths creating object pressure in the same window.
        </p>
      </div>
      <div className="gc-event-list" aria-label="Largest GC pause events">
        <div className="gc-event-list-head">
          <div>
            <h3>Largest pauses</h3>
            <p>{eventRows.length > rankedEvents.length ? `Showing ${rankedEvents.length} of ${eventRows.length} events.` : "All pause events in this range."}</p>
          </div>
          <span>Duration · cause · JVM action</span>
        </div>
        {eventRows.length === 0 ? (
          <p className="gc-empty muted">No GC pause event evidence in this range.</p>
        ) : (
          rankedEvents.map((event) => {
            const width = summary.maxDurationNs > 0 ? Math.max(4, Math.round((event.duration_ns / summary.maxDurationNs) * 100)) : 0;
            const severity = classifyPause(event.duration_ns);
            return (
              <article className={`gc-event-row gc-event-row-${severity}`} key={event.event_id}>
                <div className="gc-event-duration">
                  <strong>{formatDuration(event.duration_ns)}</strong>
                  <span>{formatTimestamp(event.event_at)}</span>
                </div>
                <div className="gc-event-body">
                  <div className="gc-event-bar" aria-hidden="true">
                    <i style={{ width: `${width}%` }} />
                  </div>
                  <div className="gc-event-meta">
                    <strong>{event.cause || "Cause unavailable"}</strong>
                    <span>{event.collector || "JVM GC"} · {event.action || event.event_type}</span>
                  </div>
                </div>
              </article>
            );
          })
        )}
      </div>
      <div className="gc-correlation-note">
        <div>
          <span>Correlation</span>
          <strong>Allocation pressure in the same profile window</strong>
        </div>
        <p>GC pauses alone show JVM stop time; allocation samples identify the Java methods most likely creating collection pressure.</p>
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

function summarizeEvents(events: JVMEventEvidence["events"], windowMs?: number) {
  const durations = events.map((event) => Math.max(0, event.duration_ns)).sort((a, b) => a - b);
  const count = durations.length;
  const totalDurationNs = durations.reduce((sum, duration) => sum + duration, 0);
  const maxDurationNs = durations[count - 1] ?? 0;
  const averageDurationNs = count > 0 ? totalDurationNs / count : 0;
  const p95DurationNs = percentile(durations, 0.95);
  const severity = classifyPause(maxDurationNs);
  const topCause = mostCommon(events.map((event) => event.cause).filter((cause): cause is string => Boolean(cause)));
  const dominantCause = topCause ? `Top cause: ${topCause}` : "Top cause unavailable";
  const eventRateLabel = windowMs && windowMs > 0 ? `${formatRate(count, windowMs)} events/min` : "Window unavailable";
  const pauseBudgetLabel = windowMs && windowMs > 0 ? `${((totalDurationNs / 1_000_000) / windowMs * 100).toFixed(2)}% of window` : `${formatDuration(averageDurationNs)} average`;
  const severityLabel = severity === "high" ? "High pause impact" : severity === "medium" ? "Moderate pause impact" : "Low pause impact";
  const interpretation = makeInterpretation({ count, severity, topCause });
  return { count, totalDurationNs, maxDurationNs, averageDurationNs, p95DurationNs, dominantCause, eventRateLabel, pauseBudgetLabel, severity, severityLabel, interpretation };
}

function rankEvents(events: JVMEvent[]) {
  return [...events].sort((a, b) => {
    const byDuration = b.duration_ns - a.duration_ns;
    if (byDuration !== 0) return byDuration;
    return Date.parse(b.event_at) - Date.parse(a.event_at);
  });
}

function percentile(sortedValues: number[], quantile: number) {
  if (sortedValues.length === 0) return 0;
  const index = Math.ceil(sortedValues.length * quantile) - 1;
  return sortedValues[Math.max(0, Math.min(sortedValues.length - 1, index))];
}

function classifyPause(ns: number) {
  if (ns >= 200_000_000) return "high";
  if (ns >= 50_000_000) return "medium";
  return "low";
}

function makeInterpretation(summary: { count: number; severity: string; topCause?: string }) {
  if (summary.count === 0) return "No GC pause evidence was collected for this scope.";
  const cause = summary.topCause ? ` Most common cause is ${summary.topCause}.` : "";
  if (summary.severity === "high") return `At least one pause exceeds 200 ms; treat this as user-visible latency evidence before chasing CPU frames.${cause}`;
  if (summary.severity === "medium") return `Pauses are measurable but below the 200 ms high-impact threshold; inspect allocation pressure and event frequency together.${cause}`;
  return `Pauses are short in this window; use allocation correlation to confirm whether object churn is still building risk.${cause}`;
}

function mostCommon(values: string[]) {
  if (values.length === 0) return undefined;
  const counts = new Map<string, number>();
  for (const value of values) counts.set(value, (counts.get(value) ?? 0) + 1);
  return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))[0]?.[0];
}

function getProfileWindow(params: URLSearchParams) {
  const start = Date.parse(params.get("start") ?? "");
  const end = Date.parse(params.get("end") ?? "");
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return undefined;
  return { start: new Date(start), end: new Date(end), durationMs: end - start };
}

function formatRate(count: number, windowMs: number) {
  const rate = count / (windowMs / 60_000);
  if (rate >= 10) return rate.toFixed(0);
  if (rate >= 1) return rate.toFixed(1);
  return rate.toFixed(2);
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "time unavailable";
  return date.toLocaleString();
}

function formatDuration(ns: number) {
  if (ns >= 1_000_000_000) return `${(ns / 1_000_000_000).toFixed(2)} s`;
  if (ns >= 1_000_000) return `${(ns / 1_000_000).toFixed(1)} ms`;
  if (ns >= 1_000) return `${(ns / 1_000).toFixed(1)} us`;
  return `${ns} ns`;
}
