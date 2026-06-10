import { useState } from "react";
import type { TargetStatus } from "../../api/types";
import { getTargetStatus } from "../../api/client";
import { useAPI } from "../../api/use-api";

const empty: TargetStatus[] = [];

export function TargetStatusView({ params, statuses }: { params: URLSearchParams; statuses?: TargetStatus[] }) {
  const [javaTargetsOnly, setJavaTargetsOnly] = useState(true);
  const loaded = useAPI(() => getTargetStatus(params), [params.toString()], empty);
  const liveQuery = statuses === undefined;
  const allStatuses = statuses ?? loaded.data ?? empty;
  const visible = javaTargetsOnly ? allStatuses.filter(isAcceptedJavaTarget) : allStatuses;
  const emptyMessage = getEmptyMessage(allStatuses, visible, javaTargetsOnly);
  return (
    <section aria-label="Target status">
      <h2>Target status</h2>
      <label className="status-filter">
        <input type="checkbox" checked={javaTargetsOnly} onChange={(event) => setJavaTargetsOnly(event.target.checked)} />
        Java targets only
      </label>
      {loaded.error && liveQuery && <p className="warning">Backend unavailable: {loaded.error}. Check the backend pod, ClickHouse schema, and Web API proxy.</p>}
      {(!liveQuery || !loaded.loading) && !loaded.error && visible.length === 0 && <p className="muted">{emptyMessage}</p>}
      <div className="table-scroll" role="region" aria-label="Target status results" tabIndex={0}>
        <table className="status-table">
          <thead>
            <tr><th>Pod</th><th>PID</th><th>Seen</th><th>State</th><th>Reason</th><th>Message</th><th>User action</th></tr>
          </thead>
          <tbody>
            {visible.map((status, index) => (
              <tr key={`${status.reason}-${index}`}>
                <td><span className="pod-name" title={status.target?.pod ?? "unknown"}>{status.target?.pod ?? "unknown"}</span></td>
                <td className="numeric">{status.target?.process_id ?? "-"}</td>
                <td className="seen-at" title={formatSeenTitle(status.status_at)}>{formatSeen(status.status_at)}</td>
                <td><span className={`state-pill ${stateClass(status.desired_state)}`}>{status.desired_state}</span></td>
                <td><span className="reason-code">{status.reason}</span></td>
                <td><span className="status-message" title={status.message}>{status.message}</span></td>
                <td><span className="status-action">{actionFor(status.reason)}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function isAcceptedJavaTarget(status: TargetStatus) {
  return status.reason === "accepted" || status.desired_state === "enabled" || status.desired_state === "temporary" || status.message.toLowerCase().includes("hotspot-compatible jvm");
}

function getEmptyMessage(allStatuses: TargetStatus[], visibleStatuses: TargetStatus[], javaTargetsOnly: boolean) {
  if (visibleStatuses.length > 0) {
    return "";
  }
  if (allStatuses.length === 0) {
    return "No matching targets for this namespace, service, Pod, JVM, or time range. Confirm the selector and profiling metadata.";
  }
  if (javaTargetsOnly) {
    const counts = summarizeReasons(allStatuses);
    return `No enabled Java targets are visible in this scope. ${counts} Uncheck \"Java targets only\" to inspect the blocked targets.`;
  }
  return "No matching targets for this namespace, service, Pod, JVM, or time range. Confirm the selector and profiling metadata.";
}

function summarizeReasons(statuses: TargetStatus[]) {
  const counts = new Map<string, number>();
  for (const status of statuses) {
    counts.set(status.reason, (counts.get(status.reason) ?? 0) + 1);
  }
  return Array.from(counts.entries())
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, 3)
    .map(([reason, count]) => `${count} ${reason}`)
    .join(", ") + ".";
}

function formatSeen(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
}

function formatSeenTitle(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toISOString();
}

function stateClass(state: string) {
  switch (state) {
    case "enabled":
    case "temporary":
      return "state-ok";
    case "unsupported":
      return "state-warn";
    case "disabled":
      return "state-muted";
    default:
      return "state-neutral";
  }
}

function actionFor(reason: string) {
  switch (reason) {
    case "accepted":
      return "Open CPU, memory, locks, or thread evidence for the same target and time range.";
    case "disabled_by_metadata":
      return "Add profiling metadata or confirm explicit disable is intended.";
    case "temporary_expired":
      return "Re-enable temporary profiling if the incident is still active.";
    case "invalid_duration":
      return "Fix duration metadata, for example 10m or 1h.";
    case "unsupported_jvm":
      return "Confirm the target is a HotSpot-compatible JVM.";
    case "profiler_conflict":
      return "Stop the other async-profiler user or skip this JVM.";
    case "attach_failed":
      return "Check collector permissions, container access, and JVM attach support.";
    case "upload_retryable":
      return "Check backend availability and network path.";
    case "upload_dropped":
      return "Check collector buffer pressure and ingestion health.";
    case "storage_rejected":
      return "Check backend validation and ClickHouse writes.";
    case "container_restarted":
      return "Open OOM investigation, then correlate restart timing with allocation, GC, and ingestion evidence.";
    case "oom_killed_seen":
      return "Open OOM investigation and inspect sampled allocation paths for object-creation pressure.";
    case "profiling_window_after_restart":
      return "Confirm the profile window starts after the restart before interpreting missing samples.";
    default:
      return "Check collector and backend logs for this target.";
  }
}
