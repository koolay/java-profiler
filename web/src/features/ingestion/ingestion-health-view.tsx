export function IngestionHealthView() {
  return (
    <section className="health-grid" aria-label="Ingestion health">
      <div><b>accepted</b><span data-source="backend-metrics">metrics</span></div>
      <div><b>duplicates</b><span data-source="backend-metrics">metrics</span></div>
      <div><b>dropped</b><span data-source="collector-metrics">metrics</span></div>
      <div><b>ttl lag</b><span data-source="backend-metrics">metrics</span></div>
    </section>
  );
}
