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
  const [tab, setTab] = useState<Tab>("cpu");
  const [namespace, setNamespace] = useState("java-profiler-qa");
  const [service, setService] = useState("jdk17-http-demo");
  const [rangeMinutes, setRangeMinutes] = useState(60);
  const params = useMemo(() => {
    const end = new Date();
    const start = new Date(end.getTime() - rangeMinutes * 60_000);
    const value = new URLSearchParams({ namespace, service, profile_type: profileTypeFor(tab), start: start.toISOString(), end: end.toISOString() });
    return value;
  }, [namespace, service, tab, rangeMinutes]);

  return (
    <div className="service-layout">
      <section className="context-strip" aria-label="Service context">
        <div className="context-fields">
          <label>
            <span>Namespace</span>
            <input value={namespace} onChange={(event) => setNamespace(event.target.value)} />
          </label>
          <label>
            <span>Service</span>
            <input value={service} onChange={(event) => setService(event.target.value)} />
          </label>
          <label>
            <span>Range</span>
            <select value={rangeMinutes} onChange={(event) => setRangeMinutes(Number(event.target.value))}>
              <option value={15}>Last 15m</option>
              <option value={30}>Last 30m</option>
              <option value={60}>Last 1h</option>
              <option value={360}>Last 6h</option>
            </select>
          </label>
          <div className="context-chip" aria-label="Timezone">
            <span>Timezone</span>
            <strong>UTC</strong>
          </div>
        </div>
        <p className="scope-note">Profiles, thread evidence, target state, and ingestion health stay here. Metric trend charts remain in Prometheus.</p>
      </section>
      <section className="diagnosis-panel">
        <div className="tab-row">
          <nav className="tabs" aria-label="Diagnosis views">
            {tabs.map((item) => (
              <button key={item} className={item === tab ? "active" : ""} onClick={() => setTab(item)}>
                {item}
              </button>
            ))}
          </nav>
        </div>
        <div className="diagnosis-content">
          {tab === "memory" && <MemoryView params={params} />}
          {tab === "cpu" && <CpuView params={params} />}
          {tab === "locks" && <LocksView params={params} />}
          {tab === "deadlocks" && <DeadlocksView params={params} />}
          {tab === "status" && <TargetStatusView params={params} />}
          {tab === "ingestion" && <IngestionHealthView />}
        </div>
      </section>
    </div>
  );
}

function profileTypeFor(tab: Tab): ProfileType {
  if (tab === "cpu") return "java_cpu_nanoseconds";
  if (tab === "locks") return "java_lock_delay_nanoseconds";
  return "java_allocation_bytes";
}
