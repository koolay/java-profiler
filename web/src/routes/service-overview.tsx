import { useMemo, useState } from "react";
import type { ProfileType } from "../api/types";
import { CpuView } from "../features/cpu/cpu-view";
import { MemoryView } from "../features/memory/memory-view";
import { LocksView } from "../features/locks/locks-view";
import { DeadlocksView } from "../features/deadlocks/deadlocks-view";
import { TargetStatusView } from "../features/status/target-status-view";
import { IngestionHealthView } from "../features/ingestion/ingestion-health-view";

const tabs = ["memory", "cpu", "locks", "deadlocks", "status", "ingestion"] as const;
type Tab = (typeof tabs)[number];

export function ServiceOverview() {
  const [tab, setTab] = useState<Tab>("memory");
  const [namespace, setNamespace] = useState("prod");
  const [service, setService] = useState("checkout");
  const [rangeMinutes, setRangeMinutes] = useState(30);
  const params = useMemo(() => {
    const end = new Date();
    const start = new Date(end.getTime() - rangeMinutes * 60_000);
    const value = new URLSearchParams({ namespace, service, profile_type: profileTypeFor(tab), start: start.toISOString(), end: end.toISOString() });
    return value;
  }, [namespace, service, tab, rangeMinutes]);

  return (
    <div className="service-layout">
      <section className="context-panel" aria-label="Service context">
        <label>
          Namespace
          <input value={namespace} onChange={(event) => setNamespace(event.target.value)} />
        </label>
        <label>
          Service
          <input value={service} onChange={(event) => setService(event.target.value)} />
        </label>
        <div className="time-grid">
          <label>
            Range
            <select value={rangeMinutes} onChange={(event) => setRangeMinutes(Number(event.target.value))}>
              <option value={15}>Last 15m</option>
              <option value={30}>Last 30m</option>
              <option value={60}>Last 1h</option>
              <option value={360}>Last 6h</option>
            </select>
          </label>
          <span>UTC</span>
        </div>
        <p className="scope-note">Metric charts stay in Prometheus. This console shows profiles, thread evidence, target state, and ingestion health.</p>
      </section>
      <section className="diagnosis-panel">
        <nav className="tabs" aria-label="Diagnosis views">
          {tabs.map((item) => (
            <button key={item} className={item === tab ? "active" : ""} onClick={() => setTab(item)}>
              {item}
            </button>
          ))}
        </nav>
        {tab === "memory" && <MemoryView params={params} />}
        {tab === "cpu" && <CpuView params={params} />}
        {tab === "locks" && <LocksView params={params} />}
        {tab === "deadlocks" && <DeadlocksView params={params} />}
        {tab === "status" && <TargetStatusView params={params} />}
        {tab === "ingestion" && <IngestionHealthView />}
      </section>
    </div>
  );
}

function profileTypeFor(tab: Tab): ProfileType {
  if (tab === "cpu") return "java_cpu_nanoseconds";
  if (tab === "locks") return "java_lock_delay_nanoseconds";
  return "java_allocation_bytes";
}
