import { getIngestionHealth } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { IngestionHealth } from "../../api/types";

const empty: IngestionHealth = {
  totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0, dropped_samples: 0, dropped_stacks: 0, truncated_batches: 0 },
  batches: [],
  partial: false,
};

type Props = {
  health?: IngestionHealth;
};

export function IngestionHealthView({ health: providedHealth }: Props = {}) {
  const { data, error, loading } = useAPI(() => getIngestionHealth(), [], empty);
  const health = providedHealth ?? data ?? empty;
  const latestLossMessage = health.batches
    .filter((batch) => batch.status === "retryable" || batch.status === "rejected")
    .sort((a, b) => (b.latest_at ?? "").localeCompare(a.latest_at ?? ""))[0]?.last_message;
  return (
    <section className="health-grid" aria-label="Ingestion health">
      {!providedHealth && loading && <p className="muted">Loading ingestion evidence.</p>}
      {!providedHealth && error && <p className="warning">Backend unavailable: {error}</p>}
      <div><b>accepted batches</b><span>{health.totals.accepted}</span></div>
      <div><b>retryable batches</b><span>{health.totals.retryable}</span></div>
      <div><b>rejected batches</b><span>{health.totals.rejected}</span></div>
      <div><b>dropped samples</b><span>{health.totals.dropped_samples}</span></div>
      <div><b>dropped stacks</b><span>{health.totals.dropped_stacks}</span></div>
      <div><b>truncated batches</b><span>{health.totals.truncated_batches}</span></div>
      {latestLossMessage && (
        <article className="event-card">
          <h3>latest retry/rejection</h3>
          <pre>{latestLossMessage}</pre>
        </article>
      )}
      {health.batches.map((batch) => (
        <article key={`${batch.batch_type}-${batch.status}`} className="event-card">
          <h3>{batch.batch_type}</h3>
          <p>{batch.status} x {batch.count}</p>
          <p>{batch.latest_at}</p>
          {batch.last_message && <pre>{batch.last_message}</pre>}
        </article>
      ))}
    </section>
  );
}
