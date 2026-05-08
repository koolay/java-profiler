import React from "react";
import { createRoot } from "react-dom/client";
import { Activity, Cpu, Database, Flame, LockKeyhole, Search } from "lucide-react";
import { ServiceOverview } from "./routes/service-overview";
import "./styles.css";

function App() {
  return (
    <main className="app-shell">
      <aside className="rail" aria-label="Java profiler navigation">
        <div className="brand">JP</div>
        <button aria-label="Service status"><Activity size={18} /></button>
        <button aria-label="CPU profiles"><Cpu size={18} /></button>
        <button aria-label="Allocation profiles"><Flame size={18} /></button>
        <button aria-label="Lock diagnosis"><LockKeyhole size={18} /></button>
        <button aria-label="Ingestion health"><Database size={18} /></button>
      </aside>
      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Kubernetes Java profiling</p>
            <h1>Service diagnosis</h1>
          </div>
          <label className="global-search">
            <Search size={16} />
            <input aria-label="Search services" placeholder="namespace / service / pod" />
          </label>
        </header>
        <ServiceOverview />
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
