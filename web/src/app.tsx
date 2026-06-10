import { useState } from "react";
import { ServiceOverview } from "./routes/service-overview";

export type DiagnosisView = "memory-pressure" | "memory" | "cpu" | "wall" | "io" | "gc" | "locks" | "deadlocks" | "status" | "ingestion";

export function App() {
  const [activeView, setActiveView] = useState<DiagnosisView>("cpu");

  return (
    <main className="app-shell">
      <ServiceOverview activeView={activeView} onViewChange={setActiveView} />
    </main>
  );
}
