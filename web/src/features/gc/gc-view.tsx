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
      <div className="gc-event-list">
        {(events?.events ?? []).length === 0 ? (
          <p className="muted">No GC pause event evidence in this range.</p>
        ) : (
          (events?.events ?? []).map((event) => (
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
        profileType="java_allocation_bytes"
        title="Allocation correlation"
        description="Allocation flamegraph is shown beside GC pauses to expose Java allocation pressure in the same selected window."
        valueLabel="Allocation"
        selfColumnLabel="Self alloc"
        totalColumnLabel="Total alloc"
      />
    </section>
  );
}

function formatDuration(ns: number) {
  if (ns >= 1_000_000_000) return `${(ns / 1_000_000_000).toFixed(2)} s`;
  if (ns >= 1_000_000) return `${(ns / 1_000_000).toFixed(1)} ms`;
  if (ns >= 1_000) return `${(ns / 1_000).toFixed(1)} us`;
  return `${ns} ns`;
}
