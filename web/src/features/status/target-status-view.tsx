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
      {loaded.error && !statuses && <p className="warning">Backend unavailable: {loaded.error}</p>}
      <table>
        <thead>
          <tr><th>Pod</th><th>PID</th><th>State</th><th>Reason</th><th>Message</th></tr>
        </thead>
        <tbody>
          {visible.map((status, index) => (
            <tr key={`${status.reason}-${index}`}>
              <td>{status.target?.pod ?? "unknown"}</td>
              <td>{status.target?.process_id ?? "-"}</td>
              <td>{status.desired_state}</td>
              <td>{status.reason}</td>
              <td>{status.message}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
