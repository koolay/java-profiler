import type { DeadlockEvent } from "../../api/types";
import { getDeadlocks } from "../../api/client";
import { useAPI } from "../../api/use-api";

const empty: DeadlockEvent[] = [];

export function DeadlocksView({ params, events }: { params: URLSearchParams; events?: DeadlockEvent[] }) {
  if (events) {
    return <DeadlocksList events={events} />;
  }
  const loaded = useAPI(() => getDeadlocks(params), [params.toString()], empty);
  return <DeadlocksList events={loaded.data ?? empty} error={loaded.error ?? undefined} />;
}

function DeadlocksList({ events, error }: { events: DeadlockEvent[]; error?: string }) {
  return (
    <section className="deadlocks deadlocks-lock-theme" aria-label="Deadlock events">
      <h2>Deadlock cycles</h2>
      {error && <p className="deadlock-warning">Backend unavailable: {error}</p>}
      {events.length === 0 && <p className="deadlock-empty">No deadlock cycles returned for this service and time range. Try a longer range or a more contended workload.</p>}
      {events.map((event) => (
        <article key={event.event_id} className="event-card deadlock-card">
          <h3>{event.cycle_id}</h3>
          <p>{event.involved_threads.join(" -> ")}</p>
          <pre className="deadlock-frames">{(event.blocking_frames ?? []).join("\n")}</pre>
        </article>
      ))}
    </section>
  );
}
