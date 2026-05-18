import type { DeadlockEvent } from "../../api/types";
import { getDeadlocks } from "../../api/client";
import { useAPI } from "../../api/use-api";

const empty: DeadlockEvent[] = [];

export function DeadlocksView({ params, events }: { params: URLSearchParams; events?: DeadlockEvent[] }) {
  const loaded = useAPI(() => getDeadlocks(params), [params.toString()], empty);
  const visible = events ?? loaded.data ?? empty;
  return (
    <section className="deadlocks" aria-label="Deadlock events">
      <h2>Deadlock cycles</h2>
      {loaded.error && !events && <p className="warning">Backend unavailable: {loaded.error}</p>}
      {visible.length === 0 && <p className="muted">No deadlock cycles returned for this service and time range. Try a longer range or a more contended workload.</p>}
      {visible.map((event) => (
        <article key={event.event_id} className="event-card">
          <h3>{event.cycle_id}</h3>
          <p>{event.involved_threads.join(" -> ")}</p>
          <pre>{(event.blocking_frames ?? []).join("\n")}</pre>
        </article>
      ))}
    </section>
  );
}
