import { getIngestionHealth } from "../../api/client";
import { useAPI } from "../../api/use-api";
import type { IngestionHealth } from "../../api/types";

const empty: IngestionHealth = {
  totals: { accepted: 0, duplicate: 0, retryable: 0, rejected: 0 },
  batches: [],
  partial: false,
};

export function IngestionHealthView() {
  const { data, error, loading } = useAPI(() => getIngestionHealth(), [], empty);
  const health = data ?? empty;
  return (
    <section className="health-grid" aria-label="Ingestion health">
      {loading && <p className="muted">Loading ingestion evidence.</p>}
      {error && <p className="warning">Backend unavailable: {error}</p>}
      <div><b>accepted</b><span>{health.totals.accepted}</span></div>
      <div><b>duplicates</b><span>{health.totals.duplicate}</span></div>
      <div><b>retryable</b><span>{health.totals.retryable}</span></div>
      <div><b>rejected</b><span>{health.totals.rejected}</span></div>
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
