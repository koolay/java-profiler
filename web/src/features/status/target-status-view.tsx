import type { TargetStatus } from "../../api/types";
import { getTargetStatus } from "../../api/client";
import { useAPI } from "../../api/use-api";

const empty: TargetStatus[] = [];

export function TargetStatusView({ params, statuses }: { params: URLSearchParams; statuses?: TargetStatus[] }) {
  const loaded = useAPI(() => getTargetStatus(params), [params.toString()], empty);
  const visible = statuses ?? loaded.data ?? empty;
  return (
    <section aria-label="Target status">
      <h2>Target status</h2>
      {loaded.error && !statuses && <p className="warning">Backend unavailable: {loaded.error}. Check the backend pod, ClickHouse schema, and Web API proxy.</p>}
      {!loaded.loading && !loaded.error && visible.length === 0 && <p className="muted">No matching targets for this namespace, service, Pod, JVM, or time range. Confirm the selector and profiling metadata.</p>}
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
                <td>{status.message}</td>
                <td>{actionFor(status.reason)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
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
      return "Open CPU, memory, locks, or thread views for the same target and time range.";
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
    default:
      return "Check collector and backend logs for this target.";
  }
}
