import { useState } from "react";
import { Activity, AlertTriangle, Cpu, Database, Flame, LockKeyhole } from "lucide-react";
import { ServiceOverview } from "./routes/service-overview";
import type { ReactNode } from "react";

export type DiagnosisView = "memory" | "cpu" | "locks" | "deadlocks" | "status" | "ingestion";

const navigationItems: Array<{
  view: DiagnosisView;
  label: string;
  icon: ReactNode;
}> = [
  { view: "status", label: "Service status", icon: <Activity size={18} /> },
  { view: "cpu", label: "CPU profiles", icon: <Cpu size={18} /> },
  { view: "memory", label: "Allocation profiles", icon: <Flame size={18} /> },
  { view: "locks", label: "Lock diagnosis", icon: <LockKeyhole size={18} /> },
  { view: "deadlocks", label: "Deadlock diagnosis", icon: <AlertTriangle size={18} /> },
  { view: "ingestion", label: "Ingestion health", icon: <Database size={18} /> },
];

export function App() {
  const [activeView, setActiveView] = useState<DiagnosisView>("cpu");

  return (
    <main className="app-shell">
      <aside className="rail" aria-label="Java profiler navigation">
        <div className="brand">JP</div>
        {navigationItems.map((item) => (
          <button
            key={item.view}
            aria-label={item.label}
            aria-pressed={item.view === activeView}
            className={item.view === activeView ? "active" : ""}
            onClick={() => setActiveView(item.view)}
            title={item.label}
            type="button"
          >
            {item.icon}
          </button>
        ))}
      </aside>
      <section className="workspace">
        <header className="topbar">
          <div className="topbar-copy">
            <p className="eyebrow">Kubernetes Java profiling</p>
            <h1>Service diagnosis</h1>
          </div>
        </header>
        <ServiceOverview activeView={activeView} onViewChange={setActiveView} />
      </section>
    </main>
  );
}
